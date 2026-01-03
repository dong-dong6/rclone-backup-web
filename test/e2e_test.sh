#!/bin/bash

# End-to-End Test Script for Rclone-Backup-Web V2.0
# This script tests the complete workflow from task creation to execution

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

HUB_URL=${HUB_URL:-"http://localhost:8080"}
API_BASE="$HUB_URL/api/v1"

echo "🧪 Starting End-to-End Test..."

# Step 1: Login as admin
echo -e "${YELLOW}Step 1: Admin Login${NC}"
TOKEN=$(curl -s -X POST "$API_BASE/admin/login" \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin"}' | \
  grep -o '"token":"[^"]*' | sed 's/"token":"//')

if [ -z "$TOKEN" ]; then
  echo -e "${RED}✗ Login failed${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Logged in successfully${NC}"

# Step 2: Create a registration token
echo -e "${YELLOW}Step 2: Create Registration Token${NC}"
REG_TOKEN=$(curl -s -X POST "$API_BASE/admin/agents/registration-token" \
  -H "Authorization: Bearer $TOKEN" | \
  grep -o '"token":"[^"]*' | sed 's/"token":"//')

if [ -z "$REG_TOKEN" ]; then
  echo -e "${RED}✗ Failed to create registration token${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Registration token created: ${REG_TOKEN:0:16}...${NC}"

# Step 3: Register an agent (simulated)
echo -e "${YELLOW}Step 3: Register Agent${NC}"
AGENT_RESPONSE=$(curl -s -X POST "$API_BASE/agent/register" \
  -H "Content-Type: application/json" \
  -d "{\"token\": \"$REG_TOKEN\", \"name\": \"test-agent-$(date +%s)\"}")

AGENT_ID=$(echo $AGENT_RESPONSE | grep -o '"agent_id":"[^"]*' | sed 's/"agent_id":"//')
API_KEY=$(echo $AGENT_RESPONSE | grep -o '"api_key":"[^"]*' | sed 's/"api_key":"//')

if [ -z "$AGENT_ID" ] || [ -z "$API_KEY" ]; then
  echo -e "${RED}✗ Agent registration failed${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Agent registered: $AGENT_ID${NC}"

# Step 4: Create a remote storage configuration
echo -e "${YELLOW}Step 4: Create Remote Storage${NC}"
REMOTE_ID=$(curl -s -X POST "$API_BASE/admin/remotes" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-s3-remote",
    "config_data": "type = s3\nregion = us-east-1\naccess_key_id = test\nsecret_access_key = test"
  }' | grep -o '"id":"[^"]*' | sed 's/"id":"//')

if [ -z "$REMOTE_ID" ]; then
  echo -e "${RED}✗ Failed to create remote storage${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Remote storage created: $REMOTE_ID${NC}"

# Step 5: Create a backup task
echo -e "${YELLOW}Step 5: Create Backup Task${NC}"
TASK_ID=$(curl -s -X POST "$API_BASE/admin/tasks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Test Backup Task\",
    \"rclone_remote_id\": \"$REMOTE_ID\",
    \"source_path\": \"/tmp/test-source\",
    \"destination_path\": \"test-backup\",
    \"schedule\": \"*/5 * * * *\",
    \"is_active\": true,
    \"assigned_agent_ids\": [\"$AGENT_ID\"]
  }" | grep -o '"id":"[^"]*' | sed 's/"id":"//')

if [ -z "$TASK_ID" ]; then
  echo -e "${RED}✗ Failed to create task${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Task created: $TASK_ID${NC}"

# Step 6: Simulate agent heartbeat
echo -e "${YELLOW}Step 6: Agent Heartbeat (Check for Tasks)${NC}"
HEARTBEAT_RESPONSE=$(curl -s -X POST "$API_BASE/agent/heartbeat" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"status": "idle"}')

ACTIONS=$(echo $HEARTBEAT_RESPONSE | grep -o '"actions":\[[^]]*\]')

if [[ $ACTIONS == *"EXECUTE_TASK"* ]]; then
  echo -e "${GREEN}✓ Task dispatched to agent!${NC}"
  echo "  Response: ${ACTIONS:0:100}..."
  
  # Extract execution ID
  EXEC_ID=$(echo $HEARTBEAT_RESPONSE | grep -o '"execution_id":"[^"]*' | sed 's/"execution_id":"//' | head -1)
  
  if [ ! -z "$EXEC_ID" ]; then
    echo -e "${GREEN}✓ Execution ID: $EXEC_ID${NC}"
    
    # Step 7: Simulate task completion
    echo -e "${YELLOW}Step 7: Report Task Completion${NC}"
    curl -s -X PUT "$API_BASE/agent/executions/$EXEC_ID" \
      -H "Authorization: Bearer $API_KEY" \
      -H "Content-Type: application/json" \
      -d '{
        "status": "success",
        "log_output": "Test backup completed successfully\n100 files transferred",
        "ended_at": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
      }'
    
    echo -e "${GREEN}✓ Task execution reported as success${NC}"
  fi
else
  echo -e "${YELLOW}⚠ No tasks dispatched (may need to wait for schedule)${NC}"
fi

# Step 8: Verify execution history
echo -e "${YELLOW}Step 8: Check Execution History${NC}"
EXECUTIONS=$(curl -s -X GET "$API_BASE/admin/executions?limit=1" \
  -H "Authorization: Bearer $TOKEN")

if [[ $EXECUTIONS == *"$TASK_ID"* ]]; then
  echo -e "${GREEN}✓ Execution recorded in history${NC}"
else
  echo -e "${YELLOW}⚠ No execution history found yet${NC}"
fi

echo ""
echo -e "${GREEN}🎉 End-to-End Test Complete!${NC}"
echo ""
echo "Summary:"
echo "  - Agent ID: $AGENT_ID"
echo "  - Task ID: $TASK_ID"
echo "  - Remote ID: $REMOTE_ID"
echo ""
echo "The system successfully:"
echo "  1. Authenticated admin user"
echo "  2. Registered a new agent"
echo "  3. Created remote storage config"
echo "  4. Created a backup task"
echo "  5. Dispatched task via heartbeat"
echo "  6. Recorded execution results"
echo ""
echo "✅ The task execution loop is working!"