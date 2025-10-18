-- Migration: 002_user_auth
-- Description: Add user authentication tables
-- Date: 2025-10-17

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) NOT NULL UNIQUE,
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255),
    role VARCHAR(50) NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'user', 'viewer')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Sessions table for token management
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,
    ip_address INET,
    user_agent TEXT,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Notification settings table
CREATE TABLE IF NOT EXISTS notification_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('email', 'webhook', 'slack', 'telegram')),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    config JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, type)
);

-- System settings table
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);

-- Apply update timestamp triggers
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notification_settings_updated_at BEFORE UPDATE ON notification_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default admin user (password: admin - should be changed on first login)
-- Password hash is for 'admin' using bcrypt
INSERT INTO users (username, email, password_hash, full_name, role) 
VALUES (
    'admin',
    'admin@localhost',
    '$2a$10$YKqm3gLz7w7gYR0M7VVxOuZKqMr.fKJT1zQvBrR5FwHsP.dZKGZGy',
    'System Administrator',
    'admin'
) ON CONFLICT (username) DO NOTHING;

-- Insert default system settings
INSERT INTO system_settings (key, value, description) VALUES
    ('maintenance_mode', '{"enabled": false}', 'System maintenance mode'),
    ('backup_retention', '{"days": 30}', 'Default backup retention period'),
    ('max_parallel_tasks', '{"count": 5}', 'Maximum parallel tasks per agent'),
    ('heartbeat_timeout', '{"seconds": 300}', 'Agent heartbeat timeout'),
    ('session_timeout', '{"hours": 24}', 'User session timeout')
ON CONFLICT (key) DO NOTHING;

-- Add comments
COMMENT ON TABLE users IS 'System users with authentication credentials';
COMMENT ON TABLE sessions IS 'Active user sessions';
COMMENT ON TABLE audit_logs IS 'Audit trail for all user actions';
COMMENT ON TABLE notification_settings IS 'User notification preferences';
COMMENT ON TABLE system_settings IS 'Global system configuration';

COMMENT ON COLUMN users.role IS 'User role: admin (full access), user (normal access), viewer (read-only)';
COMMENT ON COLUMN audit_logs.action IS 'Action performed (e.g., login, logout, create_task, delete_agent)';