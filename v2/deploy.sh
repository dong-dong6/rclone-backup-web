#!/bin/bash

# Rclone-Backup-Web V2.0 Deployment Script
# This script helps deploy the distributed backup system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${YELLOW}➜${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed"
        exit 1
    fi
    print_success "Docker is installed"
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose is not installed"
        exit 1
    fi
    print_success "Docker Compose is installed"
}

# Deploy Hub
deploy_hub() {
    print_info "Deploying Hub (Central Node)..."
    
    cd docker/hub
    
    # Copy environment file if not exists
    if [ ! -f .env ]; then
        cp .env.example .env
        print_info "Created .env file from template"
        print_info "Please edit docker/hub/.env to set your passwords and secrets"
        read -p "Press enter to continue after editing .env file..."
    fi
    
    # Start services
    docker-compose up -d
    
    print_success "Hub deployed successfully"
    print_info "Hub API: http://localhost:8080"
    print_info "Web UI: http://localhost"
}

# Register Agent
register_agent() {
    print_info "Registering a new Agent..."
    
    read -p "Enter Hub URL (e.g., http://hub.example.com): " hub_url
    read -p "Enter Agent name: " agent_name
    
    # Create registration token
    print_info "Creating registration token..."
    
    # This would normally call the API to create a token
    # For now, we'll provide instructions
    print_info "Please follow these steps:"
    echo "1. Open the Web UI at $hub_url"
    echo "2. Login with admin credentials"
    echo "3. Go to Agents page"
    echo "4. Click 'Create Registration Token'"
    echo "5. Copy the token"
    
    read -p "Enter the registration token: " token
    
    # Register agent
    response=$(curl -s -X POST "$hub_url/api/v1/agent/register" \
        -H "Content-Type: application/json" \
        -d "{\"token\": \"$token\", \"name\": \"$agent_name\"}")
    
    agent_id=$(echo $response | grep -o '"agent_id":"[^"]*' | sed 's/"agent_id":"//')
    api_key=$(echo $response | grep -o '"api_key":"[^"]*' | sed 's/"api_key":"//')
    
    if [ -z "$agent_id" ] || [ -z "$api_key" ]; then
        print_error "Failed to register agent"
        echo "Response: $response"
        exit 1
    fi
    
    print_success "Agent registered successfully"
    echo "Agent ID: $agent_id"
    echo "API Key: $api_key"
    
    # Save to agent .env file
    cd docker/agent
    cp .env.example .env
    sed -i "s|HUB_URL=.*|HUB_URL=$hub_url|" .env
    sed -i "s|AGENT_ID=.*|AGENT_ID=$agent_id|" .env
    sed -i "s|AGENT_API_KEY=.*|AGENT_API_KEY=$api_key|" .env
    
    print_success "Agent configuration saved to docker/agent/.env"
}

# Deploy Agent
deploy_agent() {
    print_info "Deploying Agent..."
    
    cd docker/agent
    
    if [ ! -f .env ]; then
        print_error "Agent not configured. Please register the agent first."
        exit 1
    fi
    
    docker-compose up -d
    
    print_success "Agent deployed successfully"
}

# Show status
show_status() {
    print_info "System Status:"
    
    echo ""
    echo "Hub Services:"
    cd docker/hub
    docker-compose ps
    
    echo ""
    echo "Agent Services:"
    cd ../agent
    if [ -f docker-compose.yml ]; then
        docker-compose ps
    else
        echo "No agents deployed"
    fi
}

# Main menu
show_menu() {
    echo ""
    echo "Rclone-Backup-Web V2.0 Deployment Tool"
    echo "======================================="
    echo "1) Deploy Hub (Central Node)"
    echo "2) Register Agent"
    echo "3) Deploy Agent"
    echo "4) Show Status"
    echo "5) Exit"
    echo ""
    read -p "Select an option: " choice
    
    case $choice in
        1)
            check_prerequisites
            deploy_hub
            ;;
        2)
            register_agent
            ;;
        3)
            deploy_agent
            ;;
        4)
            show_status
            ;;
        5)
            exit 0
            ;;
        *)
            print_error "Invalid option"
            ;;
    esac
    
    show_menu
}

# Start
cd "$(dirname "$0")"
show_menu