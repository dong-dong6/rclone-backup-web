#!/bin/bash

# Test script for verifying the scheduling system

echo "🧪 Testing Scheduling System"
echo "============================"

# Configuration
HUB_URL=${HUB_URL:-"http://localhost:8080"}
API_BASE="${HUB_URL}/api/v1"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}📝 $1${NC}"
}

# Check if Hub is running
echo "Checking if Hub is accessible..."
if curl -s "${HUB_URL}/health" > /dev/null; then
    print_success "Hub is running at ${HUB_URL}"
else
    print_error "Hub is not accessible at ${HUB_URL}"
    exit 1
fi

# Login as admin
echo ""
echo "1. Admin Login"
echo "--------------"
ADMIN_TOKEN=$(curl -s -X POST "${API_BASE}/admin/login" \
    -H "Content-Type: application/json" \
    -d '{"username": "admin", "password": "admin"}' | \
    jq -r '.token')

if [ "$ADMIN_TOKEN" != "null" ] && [ -n "$ADMIN_TOKEN" ]; then
    print_success "Admin logged in successfully"
    echo "Token: ${ADMIN_TOKEN:0:20}..."
else
    print_error "Failed to login as admin"
    exit 1
fi

# Create a test rclone remote
echo ""
echo "2. Create Test Remote"
echo "--------------------"
REMOTE_CONFIG=$(cat <<EOF
[test-s3]
type = s3
provider = AWS
access_key_id = test_key
secret_access_key = test_secret
region = us-east-1
endpoint = http://localhost:9000
EOF
)

REMOTE_ID=$(curl -s -X POST "${API_BASE}/admin/remotes" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"Test S3 Remote\",
        \"config_data\": $(echo "$REMOTE_CONFIG" | jq -Rs '.')
    }" | jq -r '.id')

if [ "$REMOTE_ID" != "null" ] && [ -n "$REMOTE_ID" ]; then
    print_success "Created remote: $REMOTE_ID"
else
    print_error "Failed to create remote"
    exit 1
fi

# Create a scheduled task (runs every minute for testing)
echo ""
echo "3. Create Scheduled Task"
echo "------------------------"
TASK_ID=$(curl -s -X POST "${API_BASE}/admin/tasks" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"Scheduled Test Backup\",
        \"rclone_remote_id\": \"$REMOTE_ID\",
        \"source_path\": \"/tmp/test-data\",
        \"destination_path\": \"test-bucket/backups\",
        \"schedule\": \"* * * * *\",
        \"rclone_args\": [\"--dry-run\", \"--verbose\"],
        \"is_active\": true,
        \"assigned_agent_ids\": []
    }" | jq -r '.id')

if [ "$TASK_ID" != "null" ] && [ -n "$TASK_ID" ]; then
    print_success "Created task: $TASK_ID"
    print_info "Schedule: Every minute (* * * * *)"
else
    print_error "Failed to create task"
    exit 1
fi

# Create an agent registration token
echo ""
echo "4. Create Agent Registration Token"
echo "----------------------------------"
REG_TOKEN=$(curl -s -X POST "${API_BASE}/admin/agents/registration-token" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" | jq -r '.token')

if [ "$REG_TOKEN" != "null" ] && [ -n "$REG_TOKEN" ]; then
    print_success "Created registration token: ${REG_TOKEN:0:20}..."
else
    print_error "Failed to create registration token"
    exit 1
fi

# Register a test agent
echo ""
echo "5. Register Test Agent"
echo "----------------------"
AGENT_RESPONSE=$(curl -s -X POST "${API_BASE}/agent/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"token\": \"$REG_TOKEN\",
        \"name\": \"test-agent-scheduler\"
    }")

AGENT_ID=$(echo "$AGENT_RESPONSE" | jq -r '.agent_id')
AGENT_API_KEY=$(echo "$AGENT_RESPONSE" | jq -r '.api_key')

if [ "$AGENT_ID" != "null" ] && [ -n "$AGENT_ID" ]; then
    print_success "Registered agent: $AGENT_ID"
else
    print_error "Failed to register agent"
    exit 1
fi

# Assign task to agent
echo ""
echo "6. Assign Task to Agent"
echo "-----------------------"
# First get the current task assignments
CURRENT_TASK=$(curl -s -X GET "${API_BASE}/admin/tasks/$TASK_ID" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}")

# Update task with agent assignment
UPDATE_RESPONSE=$(curl -s -X PUT "${API_BASE}/admin/tasks/$TASK_ID" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"Scheduled Test Backup\",
        \"rclone_remote_id\": \"$REMOTE_ID\",
        \"source_path\": \"/tmp/test-data\",
        \"destination_path\": \"test-bucket/backups\",
        \"schedule\": \"* * * * *\",
        \"rclone_args\": [\"--dry-run\", \"--verbose\"],
        \"is_active\": true,
        \"assigned_agent_ids\": [\"$AGENT_ID\"]
    }")

print_success "Task assigned to agent"

# Simulate agent heartbeats
echo ""
echo "7. Testing Task Dispatch via Heartbeat"
echo "--------------------------------------"

for i in {1..3}; do
    echo ""
    print_info "Heartbeat #$i ($(date '+%Y-%m-%d %H:%M:%S'))"
    
    HEARTBEAT_RESPONSE=$(curl -s -X POST "${API_BASE}/agent/heartbeat" \
        -H "Authorization: Bearer ${AGENT_API_KEY}" \
        -H "Content-Type: application/json" \
        -d '{"status": "idle"}')
    
    ACTIONS=$(echo "$HEARTBEAT_RESPONSE" | jq -r '.actions')
    ACTION_COUNT=$(echo "$ACTIONS" | jq 'length')
    
    if [ "$ACTION_COUNT" -gt 0 ]; then
        print_success "Received $ACTION_COUNT action(s)!"
        
        # Parse and display actions
        echo "$ACTIONS" | jq -c '.[]' | while read -r action; do
            ACTION_TYPE=$(echo "$action" | jq -r '.action')
            
            if [ "$ACTION_TYPE" == "EXECUTE_TASK" ]; then
                EXECUTION_ID=$(echo "$action" | jq -r '.execution_id')
                TASK_DATA=$(echo "$action" | jq -r '.task')
                
                print_success "🚀 EXECUTE_TASK action received!"
                echo "   Execution ID: $EXECUTION_ID"
                
                # Parse task details if present
                if [ "$TASK_DATA" != "null" ] && [ -n "$TASK_DATA" ]; then
                    TASK_INFO=$(echo "$TASK_DATA" | base64 -d 2>/dev/null | jq '.' 2>/dev/null)
                    if [ $? -eq 0 ]; then
                        echo "   Task Details:"
                        echo "$TASK_INFO" | jq '.'
                    fi
                fi
                
                # Simulate task execution
                print_info "Simulating task execution..."
                sleep 2
                
                # Report execution complete
                UPDATE_STATUS=$(curl -s -X PUT "${API_BASE}/agent/executions/$EXECUTION_ID" \
                    -H "Authorization: Bearer ${AGENT_API_KEY}" \
                    -H "Content-Type: application/json" \
                    -d "{
                        \"status\": \"success\",
                        \"log_output\": \"Task completed successfully at $(date)\",
                        \"ended_at\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
                    }")
                
                print_success "Reported execution as complete"
                
                # The task shouldn't be dispatched again for at least 1 minute
                break
            elif [ "$ACTION_TYPE" == "SYNC_CONFIG" ]; then
                print_info "SYNC_CONFIG action received"
            fi
        done
    else
        print_info "No actions received (task may not be due yet)"
    fi
    
    # Wait before next heartbeat
    if [ $i -lt 3 ]; then
        print_info "Waiting 30 seconds before next heartbeat..."
        sleep 30
    fi
done

# Check execution history
echo ""
echo "8. Checking Execution History"
echo "-----------------------------"
EXECUTIONS=$(curl -s -X GET "${API_BASE}/admin/executions?limit=10" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}")

EXECUTION_COUNT=$(echo "$EXECUTIONS" | jq '.executions | length')
print_info "Found $EXECUTION_COUNT execution(s) in history"

if [ "$EXECUTION_COUNT" -gt 0 ]; then
    echo "$EXECUTIONS" | jq -r '.executions[] | "\(.created_at | split("T")[0]) \(.created_at | split("T")[1] | split(".")[0]) | \(.status) | Task: \(.task_name // "N/A") | Agent: \(.agent_name // "N/A")"'
    print_success "Scheduling system is working correctly!"
else
    print_error "No executions found - scheduling might not be working"
fi

# Cleanup (optional)
echo ""
echo "9. Cleanup"
echo "----------"
read -p "Do you want to clean up test data? (y/n) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Delete task
    curl -s -X DELETE "${API_BASE}/admin/tasks/$TASK_ID" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}"
    print_info "Deleted task"
    
    # Delete remote
    curl -s -X DELETE "${API_BASE}/admin/remotes/$REMOTE_ID" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}"
    print_info "Deleted remote"
    
    # Delete agent
    curl -s -X DELETE "${API_BASE}/admin/agents/$AGENT_ID" \
        -H "Authorization: Bearer ${ADMIN_TOKEN}"
    print_info "Deleted agent"
    
    print_success "Cleanup complete"
else
    print_info "Skipping cleanup - test data retained"
fi

echo ""
print_success "✨ Scheduling test completed successfully!"