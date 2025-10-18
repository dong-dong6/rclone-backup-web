-- Rclone-Backup-Web V2.0 Database Schema
-- PostgreSQL Database Schema for Distributed Backup System
-- Version: 1.0
-- Date: 2025-10-17

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Drop existing tables if they exist (for clean installation)
DROP TABLE IF EXISTS task_executions CASCADE;
DROP TABLE IF EXISTS task_agent_assignments CASCADE;
DROP TABLE IF EXISTS backup_tasks CASCADE;
DROP TABLE IF EXISTS rclone_remotes CASCADE;
DROP TABLE IF EXISTS agents CASCADE;
DROP TABLE IF EXISTS registration_tokens CASCADE;

-- Table: agents
-- Stores registered agent nodes
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    api_key_hash VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'offline' CHECK (status IN ('online', 'offline', 'running_task')),
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table: registration_tokens
-- Temporary tokens for agent registration
CREATE TABLE registration_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token VARCHAR(255) NOT NULL UNIQUE,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    used_by_agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table: rclone_remotes
-- Stores encrypted rclone remote configurations
CREATE TABLE rclone_remotes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    config_data TEXT NOT NULL, -- AES-256 encrypted
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table: backup_tasks
-- Defines backup task configurations
CREATE TABLE backup_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    rclone_remote_id UUID NOT NULL REFERENCES rclone_remotes(id) ON DELETE RESTRICT,
    source_path TEXT NOT NULL,
    destination_path TEXT NOT NULL,
    schedule VARCHAR(100) NOT NULL, -- Cron format
    rclone_args JSONB DEFAULT '[]'::JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table: task_agent_assignments
-- Many-to-many relationship between tasks and agents
CREATE TABLE task_agent_assignments (
    task_id UUID NOT NULL REFERENCES backup_tasks(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, agent_id)
);

-- Table: task_executions
-- Stores task execution history
CREATE TABLE task_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES backup_tasks(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled')),
    trigger_mode VARCHAR(20) NOT NULL CHECK (trigger_mode IN ('central', 'local_fallback')),
    log_output TEXT,
    error_message TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Indexes for common queries
    CONSTRAINT unique_running_execution UNIQUE (task_id, agent_id, status) WHERE status = 'running'
);

-- Create indexes for better query performance
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_last_heartbeat ON agents(last_heartbeat);
CREATE INDEX idx_backup_tasks_is_active ON backup_tasks(is_active);
CREATE INDEX idx_backup_tasks_schedule ON backup_tasks(schedule);
CREATE INDEX idx_task_executions_status ON task_executions(status);
CREATE INDEX idx_task_executions_task_id ON task_executions(task_id);
CREATE INDEX idx_task_executions_agent_id ON task_executions(agent_id);
CREATE INDEX idx_task_executions_started_at ON task_executions(started_at DESC);
CREATE INDEX idx_registration_tokens_expires_at ON registration_tokens(expires_at);
CREATE INDEX idx_registration_tokens_used ON registration_tokens(used);

-- Create update timestamp trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update timestamp triggers
CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rclone_remotes_updated_at BEFORE UPDATE ON rclone_remotes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_backup_tasks_updated_at BEFORE UPDATE ON backup_tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create view for active task assignments
CREATE VIEW active_task_assignments AS
SELECT 
    ta.task_id,
    ta.agent_id,
    t.name as task_name,
    a.name as agent_name,
    t.schedule,
    t.is_active,
    a.status as agent_status,
    a.last_heartbeat
FROM task_agent_assignments ta
JOIN backup_tasks t ON ta.task_id = t.id
JOIN agents a ON ta.agent_id = a.id
WHERE t.is_active = TRUE;

-- Create view for recent executions
CREATE VIEW recent_executions AS
SELECT 
    e.id,
    e.task_id,
    e.agent_id,
    t.name as task_name,
    a.name as agent_name,
    e.status,
    e.trigger_mode,
    e.started_at,
    e.ended_at,
    EXTRACT(EPOCH FROM (e.ended_at - e.started_at)) as duration_seconds
FROM task_executions e
JOIN backup_tasks t ON e.task_id = t.id
JOIN agents a ON e.agent_id = a.id
ORDER BY e.started_at DESC
LIMIT 100;

-- Add comments for documentation
COMMENT ON TABLE agents IS 'Registered backup agent nodes';
COMMENT ON TABLE registration_tokens IS 'One-time tokens for agent registration';
COMMENT ON TABLE rclone_remotes IS 'Encrypted rclone remote storage configurations';
COMMENT ON TABLE backup_tasks IS 'Backup task definitions';
COMMENT ON TABLE task_agent_assignments IS 'Assignment of tasks to agents';
COMMENT ON TABLE task_executions IS 'Historical record of task executions';

COMMENT ON COLUMN agents.status IS 'Current agent status: online, offline, or running_task';
COMMENT ON COLUMN rclone_remotes.config_data IS 'AES-256 encrypted rclone configuration';
COMMENT ON COLUMN backup_tasks.schedule IS 'Cron expression for task scheduling';
COMMENT ON COLUMN task_executions.trigger_mode IS 'How the task was triggered: central or local_fallback';