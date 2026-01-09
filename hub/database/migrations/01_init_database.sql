-- ============================================
-- Rclone-Backup-Web V2.0 完整数据库架构
-- 此文件包含所有架构定义
-- ============================================

-- 创建 UUID 扩展（如果不存在）
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- 核心表
-- ============================================

-- Agents 表：存储备份代理信息
CREATE TABLE IF NOT EXISTS agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),              -- 代理唯一标识符
    name VARCHAR(255) NOT NULL,                                  -- 代理名称
    api_key_hash VARCHAR(255) NOT NULL UNIQUE,                   -- API 密钥哈希值（用于认证）
    status VARCHAR(20) NOT NULL DEFAULT 'offline' 
        CHECK (status IN ('online', 'offline', 'running_task')), -- 代理当前状态
    last_heartbeat TIMESTAMP WITH TIME ZONE,                     -- 最后一次心跳时间
    current_task UUID,                                           -- 当前执行的任务 ID
    version VARCHAR(50),                                         -- 代理软件版本
    is_local BOOLEAN NOT NULL DEFAULT FALSE,                     -- 是否为本地Agent（不可删除）
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),  -- 记录创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()   -- 记录更新时间
);

-- 创建索引用于快速过滤
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_last_heartbeat ON agents(last_heartbeat);

-- Agent metrics历史数据表：存储每次心跳收集的指标快照
CREATE TABLE IF NOT EXISTS agent_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    hostname VARCHAR(255),
    platform VARCHAR(100),
    agent_version VARCHAR(100),

    cpu_usage DECIMAL(5,2),

    memory_total BIGINT,
    memory_used BIGINT,
    memory_usage DECIMAL(5,2),
    swap_total BIGINT,
    swap_used BIGINT,

    disk_total BIGINT,
    disk_used BIGINT,
    disk_usage DECIMAL(5,2),

    network_rx_bytes BIGINT,
    network_tx_bytes BIGINT,
    network_rx_rate BIGINT,
    network_tx_rate BIGINT,

    tcp_connections INTEGER,
    udp_connections INTEGER,

    process_count INTEGER,

    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_metrics_agent_time ON agent_metrics(agent_id, recorded_at DESC);
CREATE INDEX idx_agent_metrics_recorded_at ON agent_metrics(recorded_at);

-- Rclone 远程配置表：存储 rclone 远程配置
CREATE TABLE IF NOT EXISTS rclone_remotes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),              -- 远程配置唯一标识符
    name VARCHAR(255) NOT NULL UNIQUE,                           -- 远程名称（必须唯一）
    config_data TEXT NOT NULL,                                   -- 加密的 rclone 配置数据
    type VARCHAR(50),                                            -- 远程类型（如 s3, gdrive, sftp）
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),  -- 记录创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()   -- 记录更新时间
);

-- 备份任务表：定义定时备份作业
CREATE TABLE IF NOT EXISTS backup_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),              -- 任务唯一标识符
    name VARCHAR(255) NOT NULL,                                  -- 任务名称
    rclone_remote_id UUID NOT NULL 
        REFERENCES rclone_remotes(id) ON DELETE CASCADE,         -- 关联的 rclone 远程配置
    source_type VARCHAR(20) NOT NULL DEFAULT 'path'
        CHECK (source_type IN ('path', 'database')),             -- 备份源类型：path=文件路径，database=数据库
    source_path TEXT NOT NULL,                                   -- 备份源路径（source_type=path 时必填；sqlite 可复用为 DB 文件路径）
    db_engine VARCHAR(20)
        CHECK (db_engine IN ('postgres', 'mysql', 'sqlite')),    -- 数据库类型（source_type=database）
    db_host TEXT,                                                -- 数据库主机（postgres/mysql）
    db_port INTEGER,                                             -- 数据库端口（postgres/mysql）
    db_user TEXT,                                                -- 数据库用户（postgres/mysql）
    db_name TEXT,                                                -- 数据库名（postgres/mysql）
    db_password TEXT,                                            -- 数据库口令（Hub 端加密存储）
    db_path TEXT,                                                -- SQLite 数据库文件路径（source_type=database 且 db_engine=sqlite）
    destination_path TEXT NOT NULL,                              -- 远程目标路径
    schedule VARCHAR(100) NOT NULL,                              -- Cron 表达式（定时调度）
    rclone_args JSONB DEFAULT '[]'::JSONB,                       -- 额外的 rclone 命令参数
    is_active BOOLEAN NOT NULL DEFAULT true,                     -- 任务是否启用
    backup_mode VARCHAR(20) NOT NULL DEFAULT 'sync' 
        CHECK (backup_mode IN ('sync', 'archive')),              -- 备份模式：sync=镜像同步，archive=压缩归档
    archive_format VARCHAR(20) NOT NULL DEFAULT 'tar.gz'
        CHECK (archive_format IN ('tar.gz', 'zip', '7z')),        -- archive 模式压缩格式
    encryption_enabled BOOLEAN NOT NULL DEFAULT false,           -- 是否启用加密（rclone crypt）
    encryption_password TEXT,                                    -- 任务加密口令（Hub 端加密存储）
    encryption_password2 TEXT,                                   -- 任务加密盐口令（Hub 端加密存储）
    retention_days INTEGER DEFAULT 30,                           -- 备份数据保留天数
    max_retention INTEGER,                                       -- archive 模式最大留存份数（超出将删除旧备份）
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),  -- 记录创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()   -- 记录更新时间
);

-- 创建任务查询索引
CREATE INDEX idx_backup_tasks_active ON backup_tasks(is_active);
CREATE INDEX idx_backup_tasks_schedule ON backup_tasks(schedule);

-- 任务与代理分配表：任务和代理的多对多关系
CREATE TABLE IF NOT EXISTS task_agent_assignments (
    task_id UUID NOT NULL 
        REFERENCES backup_tasks(id) ON DELETE CASCADE,           -- 备份任务引用
    agent_id UUID NOT NULL 
        REFERENCES agents(id) ON DELETE CASCADE,                 -- 代理引用
    priority INTEGER DEFAULT 0,                                  -- 执行优先级（值越高优先级越高）
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),  -- 分配创建时间
    PRIMARY KEY (task_id, agent_id)
);

-- 任务执行历史表：记录任务执行情况
CREATE TABLE IF NOT EXISTS task_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),              -- 执行记录唯一标识符
    task_id UUID NOT NULL 
        REFERENCES backup_tasks(id) ON DELETE CASCADE,           -- 备份任务引用
    agent_id UUID NOT NULL 
        REFERENCES agents(id) ON DELETE CASCADE,                 -- 执行任务的代理
    status VARCHAR(20) NOT NULL DEFAULT 'pending' 
        CHECK (status IN ('pending', 'running', 'success', 
                          'failed', 'cancelled')),               -- 执行状态
    trigger_mode VARCHAR(20) NOT NULL DEFAULT 'scheduled' 
        CHECK (trigger_mode IN ('scheduled', 'manual', 
                                'retry', 'local_fallback')),     -- 触发方式
    log_output TEXT,                                             -- 完整执行日志
    error_message TEXT,                                          -- 错误信息（失败时）
    bytes_transferred BIGINT,                                    -- 传输的总字节数
    files_transferred INTEGER,                                   -- 传输的文件数量
    duration_seconds INTEGER,                                    -- 执行耗时（秒）
    started_at TIMESTAMP WITH TIME ZONE,                         -- 执行开始时间
    ended_at TIMESTAMP WITH TIME ZONE,                           -- 执行结束时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()   -- 记录创建时间
);

-- 创建执行记录查询索引
CREATE INDEX idx_task_executions_task_id ON task_executions(task_id);
CREATE INDEX idx_task_executions_agent_id ON task_executions(agent_id);
CREATE INDEX idx_task_executions_status ON task_executions(status);
CREATE INDEX idx_task_executions_created_at ON task_executions(created_at DESC);

-- 注册令牌表：代理注册使用的一次性令牌
CREATE TABLE IF NOT EXISTS registration_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),              -- 令牌唯一标识符
    token VARCHAR(255) NOT NULL UNIQUE,                          -- 注册令牌字符串
    used BOOLEAN DEFAULT FALSE NOT NULL,                         -- 令牌是否已使用
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,                -- 令牌过期时间
    used_at TIMESTAMP WITH TIME ZONE,                            -- 令牌使用时间
    used_by_agent_id UUID REFERENCES agents(id),                 -- 使用此令牌的代理
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()   -- 令牌创建时间
);

CREATE INDEX idx_registration_tokens_token ON registration_tokens(token);
CREATE INDEX idx_registration_tokens_expires_at ON registration_tokens(expires_at);

-- ============================================
-- 用户认证与授权
-- ============================================

-- 用户表：系统用户账户
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),              -- 用户唯一标识符
    username VARCHAR(100) NOT NULL UNIQUE,                       -- 登录用户名
    email VARCHAR(255) UNIQUE,                                   -- 用户邮箱地址
    password_hash VARCHAR(255) NOT NULL,                         -- Bcrypt 加密的密码哈希
    full_name VARCHAR(255),                                      -- 用户全名
    role VARCHAR(50) DEFAULT 'user',                             -- 用户角色（admin, user 等）
    is_active BOOLEAN DEFAULT true,                              -- 账户是否激活
    is_admin BOOLEAN DEFAULT false,                              -- 是否拥有管理员权限
    last_login TIMESTAMP WITH TIME ZONE,                         -- 最后登录时间
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),  -- 账户创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()   -- 账户更新时间
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

-- 会话表：活跃的用户会话
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),              -- 会话唯一标识符
    user_id UUID NOT NULL 
        REFERENCES users(id) ON DELETE CASCADE,                  -- 用户引用
    token_hash VARCHAR(255) NOT NULL UNIQUE,                     -- 会话令牌哈希值
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,                -- 会话过期时间
    ip_address INET,                                             -- 客户端 IP 地址
    user_agent TEXT,                                             -- 客户端 User-Agent 字符串
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()   -- 会话创建时间
);

CREATE INDEX idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

-- 审计日志表：系统操作审计记录
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),              -- 日志唯一标识符
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,        -- 执行操作的用户
    action VARCHAR(100) NOT NULL,                                -- 操作类型（如 create, update, delete）
    resource_type VARCHAR(50),                                   -- 受影响的资源类型
    resource_id UUID,                                            -- 受影响的资源 ID
    details JSONB,                                               -- 操作详情（JSON 格式）
    ip_address INET,                                             -- 客户端 IP 地址
    user_agent TEXT,                                             -- 客户端 User-Agent 字符串
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()   -- 日志记录时间
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- 通知设置表：用户通知偏好
CREATE TABLE IF NOT EXISTS notification_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),              -- 设置唯一标识符
    user_id UUID NOT NULL 
        REFERENCES users(id) ON DELETE CASCADE,                  -- 用户引用
    type VARCHAR(50) NOT NULL,                                   -- 通知类型（email, webhook 等）
    enabled BOOLEAN DEFAULT true,                                -- 是否启用通知
    config JSONB,                                                -- 通知配置（JSON 格式）
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),  -- 设置创建时间
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),  -- 设置更新时间
    UNIQUE(user_id, type)
);

-- 系统设置表：全局系统配置
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(100) PRIMARY KEY,                                -- 设置键名
    value TEXT NOT NULL,                                         -- 设置值
    description TEXT,                                            -- 可读的描述信息
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()   -- 设置更新时间
);

-- ============================================
-- 函数和触发器
-- ============================================

-- 自动更新 updated_at 时间戳的函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 为包含 updated_at 的表应用更新触发器
CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rclone_remotes_updated_at BEFORE UPDATE ON rclone_remotes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_backup_tasks_updated_at BEFORE UPDATE ON backup_tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notification_settings_updated_at BEFORE UPDATE ON notification_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- 当注册令牌被标记为已使用时自动设置 used_at 的函数
CREATE OR REPLACE FUNCTION update_used_at()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.used = TRUE AND OLD.used = FALSE AND NEW.used_at IS NULL THEN
        NEW.used_at = NOW();
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 当 used 设为 true 时更新 used_at 的触发器
CREATE TRIGGER update_registration_token_used_at
    BEFORE UPDATE ON registration_tokens
    FOR EACH ROW EXECUTE FUNCTION update_used_at();

-- ============================================
-- 初始数据
-- ============================================

-- 注意：管理员用户将在部署时使用随机密码创建
-- 参见 deploy.sh 了解初始管理员账户设置

-- 插入默认系统设置
INSERT INTO system_settings (key, value, description) VALUES
    ('backup.default_retention_days', '30', '备份数据默认保留天数'),
    ('agent.heartbeat_interval', '30', '代理心跳间隔（秒）'),
    ('agent.offline_threshold', '120', '代理离线判定阈值（秒）'),
    ('execution.timeout', '7200', '任务执行默认超时时间（秒）'),
    ('execution.log_retention_days', '30', '执行日志保留天数'),
    ('metrics.retention_hours', '168', '监控数据保留时间（小时），默认 7 天'),
    ('metrics.sample_interval', '30', '监控数据采样间隔（秒）')
ON CONFLICT (key) DO NOTHING;

-- ============================================
-- 权限设置
-- ============================================

-- 授予权限（根据你的 PostgreSQL 配置调整）
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO rclone;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO rclone;

-- ============================================
-- 验证
-- ============================================

-- 列出所有表（用于验证）
DO $$
BEGIN
    RAISE NOTICE '数据库初始化成功完成';
    RAISE NOTICE '已创建表：agents, rclone_remotes, backup_tasks, task_agent_assignments, task_executions, registration_tokens, users, sessions, audit_logs, notification_settings, system_settings';
END $$;
