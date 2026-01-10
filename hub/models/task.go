package models

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BackupTask struct {
	ID                uuid.UUID       `json:"id"`
	Name              string          `json:"name"`
	RcloneRemoteID    uuid.UUID       `json:"rclone_remote_id"`
	SourceType        string          `json:"source_type"`
	SourcePath        string          `json:"source_path"`
	DBEngine          *string         `json:"db_engine,omitempty"`
	DBHost            *string         `json:"db_host,omitempty"`
	DBPort            *int            `json:"db_port,omitempty"`
	DBUser            *string         `json:"db_user,omitempty"`
	DBName            *string         `json:"db_name,omitempty"`
	DBDumpMode        *string         `json:"db_dump_mode,omitempty"`
	DBPath            *string         `json:"db_path,omitempty"`
	DestinationPath   string          `json:"destination_path"`
	Schedule          string          `json:"schedule"`
	RcloneArgs        json.RawMessage `json:"rclone_args"`
	IsActive          bool            `json:"is_active"`
	BackupMode        string          `json:"backup_mode"`
	ArchiveFormat     string          `json:"archive_format,omitempty"`
	EncryptionEnabled bool            `json:"encryption_enabled"`
	RetentionDays     *int            `json:"retention_days,omitempty"`
	MaxRetention      *int            `json:"max_retention,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`

	EncryptionPasswordEnc  *string `json:"-"`
	EncryptionPassword2Enc *string `json:"-"`
	DBPasswordEnc          *string `json:"-"`

	// Additional fields from joins
	RemoteName     string      `json:"remote_name,omitempty"`
	AssignedAgents []uuid.UUID `json:"assigned_agents,omitempty"`
}

type TaskModel struct {
	db *pgxpool.Pool
}

func NewTaskModel(db *pgxpool.Pool) *TaskModel {
	return &TaskModel{db: db}
}

// Create creates a new backup task
func (m *TaskModel) Create(ctx context.Context, task *BackupTask) error {
	task.ID = uuid.New()
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	if task.RcloneArgs == nil {
		task.RcloneArgs = json.RawMessage("[]")
	}
	if strings.TrimSpace(task.SourceType) == "" {
		task.SourceType = "path"
	}
	if strings.TrimSpace(task.BackupMode) == "" {
		task.BackupMode = "sync"
	}
	if strings.TrimSpace(task.ArchiveFormat) == "" {
		task.ArchiveFormat = "tar.gz"
	}

	query := `
				INSERT INTO backup_tasks (
					id, name, rclone_remote_id,
					source_type, source_path, db_engine, db_host, db_port, db_user, db_name, db_dump_mode, db_password, db_path,
					destination_path,
					schedule, rclone_args, is_active,
					backup_mode, archive_format, encryption_enabled, encryption_password, encryption_password2,
					retention_days, max_retention, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
				RETURNING id, created_at, updated_at
			`

	err := m.db.QueryRow(ctx, query,
		task.ID,
		task.Name,
		task.RcloneRemoteID,
		task.SourceType,
		task.SourcePath,
		task.DBEngine,
		task.DBHost,
		task.DBPort,
		task.DBUser,
		task.DBName,
		task.DBDumpMode,
		task.DBPasswordEnc,
		task.DBPath,
		task.DestinationPath,
		task.Schedule,
		task.RcloneArgs,
		task.IsActive,
		task.BackupMode,
		task.ArchiveFormat,
		task.EncryptionEnabled,
		task.EncryptionPasswordEnc,
		task.EncryptionPassword2Enc,
		task.RetentionDays,
		task.MaxRetention,
		task.CreatedAt,
		task.UpdatedAt,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)

	return err
}

// GetByID retrieves a task by ID
func (m *TaskModel) GetByID(ctx context.Context, id uuid.UUID) (*BackupTask, error) {
	task := &BackupTask{}
	query := `
				SELECT 
					t.id, t.name, t.rclone_remote_id,
					t.source_type, t.source_path, t.db_engine, t.db_host, t.db_port, t.db_user, t.db_name, t.db_dump_mode, t.db_password, t.db_path,
					t.destination_path,
					t.schedule, t.rclone_args, t.is_active,
					t.backup_mode, t.archive_format, t.encryption_enabled, t.encryption_password, t.encryption_password2,
					t.retention_days, t.max_retention, t.created_at, t.updated_at,
					r.name as remote_name
				FROM backup_tasks t
			LEFT JOIN rclone_remotes r ON t.rclone_remote_id = r.id
			WHERE t.id = $1
	`

	err := m.db.QueryRow(ctx, query, id).Scan(
		&task.ID,
		&task.Name,
		&task.RcloneRemoteID,
		&task.SourceType,
		&task.SourcePath,
		&task.DBEngine,
		&task.DBHost,
		&task.DBPort,
		&task.DBUser,
		&task.DBName,
		&task.DBDumpMode,
		&task.DBPasswordEnc,
		&task.DBPath,
		&task.DestinationPath,
		&task.Schedule,
		&task.RcloneArgs,
		&task.IsActive,
		&task.BackupMode,
		&task.ArchiveFormat,
		&task.EncryptionEnabled,
		&task.EncryptionPasswordEnc,
		&task.EncryptionPassword2Enc,
		&task.RetentionDays,
		&task.MaxRetention,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.RemoteName,
	)

	if err != nil {
		return nil, err
	}

	// Get assigned agents
	task.AssignedAgents, _ = m.GetAssignedAgents(ctx, task.ID)

	return task, nil
}

// List retrieves all tasks
func (m *TaskModel) List(ctx context.Context) ([]*BackupTask, error) {
	query := `
				SELECT 
					t.id, t.name, t.rclone_remote_id,
					t.source_type, t.source_path, t.db_engine, t.db_host, t.db_port, t.db_user, t.db_name, t.db_dump_mode, t.db_password, t.db_path,
					t.destination_path,
					t.schedule, t.rclone_args, t.is_active,
					t.backup_mode, t.archive_format, t.encryption_enabled, t.encryption_password, t.encryption_password2,
					t.retention_days, t.max_retention, t.created_at, t.updated_at,
					r.name as remote_name
				FROM backup_tasks t
			LEFT JOIN rclone_remotes r ON t.rclone_remote_id = r.id
			ORDER BY t.created_at DESC
		`

	rows, err := m.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []*BackupTask{}
	for rows.Next() {
		task := &BackupTask{}
		err := rows.Scan(
			&task.ID,
			&task.Name,
			&task.RcloneRemoteID,
			&task.SourceType,
			&task.SourcePath,
			&task.DBEngine,
			&task.DBHost,
			&task.DBPort,
			&task.DBUser,
			&task.DBName,
			&task.DBDumpMode,
			&task.DBPasswordEnc,
			&task.DBPath,
			&task.DestinationPath,
			&task.Schedule,
			&task.RcloneArgs,
			&task.IsActive,
			&task.BackupMode,
			&task.ArchiveFormat,
			&task.EncryptionEnabled,
			&task.EncryptionPasswordEnc,
			&task.EncryptionPassword2Enc,
			&task.RetentionDays,
			&task.MaxRetention,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.RemoteName,
		)
		if err != nil {
			return nil, err
		}

		// Get assigned agents for each task
		task.AssignedAgents, _ = m.GetAssignedAgents(ctx, task.ID)

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// Update updates a backup task
func (m *TaskModel) Update(ctx context.Context, task *BackupTask) error {
	task.UpdatedAt = time.Now()

	query := `
				UPDATE backup_tasks SET
					name = $2,
					rclone_remote_id = $3,
					source_type = $4,
					source_path = $5,
					db_engine = $6,
					db_host = $7,
					db_port = $8,
					db_user = $9,
					db_name = $10,
					db_dump_mode = $11,
					db_password = $12,
					db_path = $13,
					destination_path = $14,
					schedule = $15,
					rclone_args = $16,
					is_active = $17,
					backup_mode = $18,
					archive_format = $19,
					encryption_enabled = $20,
					encryption_password = $21,
					encryption_password2 = $22,
					retention_days = $23,
					max_retention = $24,
					updated_at = $25
				WHERE id = $1
			`

	_, err := m.db.Exec(ctx, query,
		task.ID,
		task.Name,
		task.RcloneRemoteID,
		task.SourceType,
		task.SourcePath,
		task.DBEngine,
		task.DBHost,
		task.DBPort,
		task.DBUser,
		task.DBName,
		task.DBDumpMode,
		task.DBPasswordEnc,
		task.DBPath,
		task.DestinationPath,
		task.Schedule,
		task.RcloneArgs,
		task.IsActive,
		task.BackupMode,
		task.ArchiveFormat,
		task.EncryptionEnabled,
		task.EncryptionPasswordEnc,
		task.EncryptionPassword2Enc,
		task.RetentionDays,
		task.MaxRetention,
		task.UpdatedAt,
	)

	return err
}

// Delete deletes a backup task
func (m *TaskModel) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM backup_tasks WHERE id = $1`
	_, err := m.db.Exec(ctx, query, id)
	return err
}

// AssignAgent assigns a task to an agent
func (m *TaskModel) AssignAgent(ctx context.Context, taskID, agentID uuid.UUID) error {
	query := `
		INSERT INTO task_agent_assignments (task_id, agent_id)
		VALUES ($1, $2)
		ON CONFLICT (task_id, agent_id) DO NOTHING
	`

	_, err := m.db.Exec(ctx, query, taskID, agentID)
	return err
}

// UnassignAgent unassigns a task from an agent
func (m *TaskModel) UnassignAgent(ctx context.Context, taskID, agentID uuid.UUID) error {
	query := `
		DELETE FROM task_agent_assignments
		WHERE task_id = $1 AND agent_id = $2
	`

	_, err := m.db.Exec(ctx, query, taskID, agentID)
	return err
}

// GetAssignedAgents gets all agents assigned to a task
func (m *TaskModel) GetAssignedAgents(ctx context.Context, taskID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT agent_id
		FROM task_agent_assignments
		WHERE task_id = $1
	`

	rows, err := m.db.Query(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	agents := []uuid.UUID{}
	for rows.Next() {
		var agentID uuid.UUID
		if err := rows.Scan(&agentID); err != nil {
			return nil, err
		}
		agents = append(agents, agentID)
	}

	return agents, nil
}

// GetAgentTasks gets all tasks assigned to an agent
func (m *TaskModel) GetAgentTasks(ctx context.Context, agentID uuid.UUID) ([]*BackupTask, error) {
	query := `
			SELECT 
				t.id, t.name, t.rclone_remote_id,
				t.source_type, t.source_path, t.db_engine, t.db_host, t.db_port, t.db_user, t.db_name, t.db_dump_mode, t.db_password, t.db_path,
				t.destination_path,
				t.schedule, t.rclone_args, t.is_active,
				t.backup_mode, t.archive_format, t.encryption_enabled, t.encryption_password, t.encryption_password2,
				t.max_retention, t.created_at, t.updated_at
			FROM backup_tasks t
		INNER JOIN task_agent_assignments ta ON t.id = ta.task_id
		WHERE ta.agent_id = $1 AND t.is_active = true
		ORDER BY t.created_at DESC
	`

	rows, err := m.db.Query(ctx, query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []*BackupTask{}
	for rows.Next() {
		task := &BackupTask{}
		err := rows.Scan(
			&task.ID,
			&task.Name,
			&task.RcloneRemoteID,
			&task.SourceType,
			&task.SourcePath,
			&task.DBEngine,
			&task.DBHost,
			&task.DBPort,
			&task.DBUser,
			&task.DBName,
			&task.DBDumpMode,
			&task.DBPasswordEnc,
			&task.DBPath,
			&task.DestinationPath,
			&task.Schedule,
			&task.RcloneArgs,
			&task.IsActive,
			&task.BackupMode,
			&task.ArchiveFormat,
			&task.EncryptionEnabled,
			&task.EncryptionPasswordEnc,
			&task.EncryptionPassword2Enc,
			&task.MaxRetention,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}
