# Rclone-Backup-Web V2.0 Makefile

.PHONY: help build run test clean deploy-hub deploy-agent dev

# Variables
HUB_IMAGE := rclone-hub:v2.0.0
AGENT_IMAGE := rclone-agent:v2.0.0
WEB_IMAGE := rclone-web:v2.0.0

help: ## Show this help message
	@echo "Rclone-Backup-Web V2.0 - Distributed Backup System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Development
dev-hub: ## Run Hub API in development mode
	cd hub && go run .

dev-agent: ## Run Agent in development mode
	cd agent && go run .

dev-web: ## Run Web UI in development mode
	cd hub/web && npm install && npm run dev

dev-db: ## Start development database
	docker run -d --name rclone-dev-db \
		-e POSTGRES_PASSWORD=password \
		-e POSTGRES_DB=rclone_backup \
		-p 5432:5432 \
		postgres:15-alpine

# Build
build-hub: ## Build Hub Docker image
	cd hub && docker build -t $(HUB_IMAGE) .

build-agent: ## Build Agent Docker image
	cd agent && docker build -t $(AGENT_IMAGE) .

build-web: ## Build Web UI
	cd hub/web && npm install && npm run build

build: build-hub build-agent build-web ## Build all components

# Testing
test-hub: ## Run Hub tests
	cd hub && go test ./...

test-agent: ## Run Agent tests
	cd agent && go test ./...

test: test-hub test-agent ## Run all tests

# Database
db-migrate: ## Run database migrations
	docker exec -i rclone-hub-db psql -U postgres -d rclone_backup < database/schema.sql

db-reset: ## Reset database
	docker exec -i rclone-hub-db psql -U postgres -c "DROP DATABASE IF EXISTS rclone_backup; CREATE DATABASE rclone_backup;"
	$(MAKE) db-migrate

# Deployment
deploy-hub: ## Deploy Hub using Docker Compose
	cd docker/hub && docker-compose up -d

deploy-agent: ## Deploy Agent using Docker Compose
	cd docker/agent && docker-compose up -d

stop-hub: ## Stop Hub services
	cd docker/hub && docker-compose down

stop-agent: ## Stop Agent services
	cd docker/agent && docker-compose down

logs-hub: ## View Hub logs
	cd docker/hub && docker-compose logs -f

logs-agent: ## View Agent logs
	cd docker/agent && docker-compose logs -f

# Cleanup
clean-containers: ## Remove all containers
	docker-compose -f docker/hub/docker-compose.yml down -v
	docker-compose -f docker/agent/docker-compose.yml down -v

clean-images: ## Remove built images
	docker rmi $(HUB_IMAGE) $(AGENT_IMAGE) $(WEB_IMAGE) 2>/dev/null || true

clean: clean-containers clean-images ## Clean everything

# Utilities
gen-key: ## Generate encryption key
	@openssl rand -base64 32

gen-jwt-secret: ## Generate JWT secret
	@openssl rand -base64 64

init-env: ## Initialize environment files
	@if [ ! -f docker/hub/.env ]; then \
		cp docker/hub/.env.example docker/hub/.env; \
		echo "Created docker/hub/.env - Please edit with your configuration"; \
	fi
	@if [ ! -f docker/agent/.env ]; then \
		cp docker/agent/.env.example docker/agent/.env; \
		echo "Created docker/agent/.env - Please edit with your configuration"; \
	fi

# Quick start
quick-start: init-env deploy-hub ## Quick start Hub deployment
	@echo ""
	@echo "Hub deployed successfully!"
	@echo "Access the Web UI at: http://localhost"
	@echo "API endpoint: http://localhost:8080"
	@echo ""
	@echo "Default credentials: admin/admin"
	@echo "Please change the password after first login"

# Status
status: ## Show services status
	@echo "Hub Services:"
	@cd docker/hub && docker-compose ps
	@echo ""
	@echo "Agent Services:"
	@cd docker/agent && docker-compose ps 2>/dev/null || echo "No agents running"