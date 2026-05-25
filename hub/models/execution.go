package models

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	TriggerModeScheduled     = "scheduled"
	TriggerModeManual        = "manual"
	TriggerModeRetry         = "retry"
	TriggerModeLocalFallback = "local_fallback"
)

var validTriggerModes = map[string]struct{}{
	TriggerModeScheduled:     {},
	TriggerModeManual:        {},
	TriggerModeRetry:         {},
	TriggerModeLocalFallback: {},
}

type TaskExecution struct {
	ID               uuid.UUID  `json:"id"`
	TaskID           uuid.UUID  `json:"task_id"`
	AgentID          uuid.UUID  `json:"agent_id"`
	Status           string     `json:"status"`
	TriggerMode      string     `json:"trigger_mode"`
	LogOutput        string     `json:"log_output,omitempty"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	BytesTransferred *int64     `json:"bytes_transferred,omitempty"`
	FilesTransferred *int       `json:"files_transferred,omitempty"`
	DurationSeconds  *int       `json:"duration_seconds,omitempty"`
	StartedAt        *time.Time `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at"`
	CreatedAt        time.Time  `json:"created_at"`

	// Additional fields from joins
	TaskName  string `json:"task_name,omitempty"`
	AgentName string `json:"agent_name,omitempty"`
}

type ExecutionModel struct {
	db *pgxpool.Pool
}

func NewExecutionModel(db *pgxpool.Pool) *ExecutionModel {
	return &ExecutionModel{db: db}
}

// NormalizeTriggerMode validates and normalizes trigger mode values
func NormalizeTriggerMode(triggerMode string) (string, error) {
	if triggerMode == "" {
		triggerMode = TriggerModeScheduled
	}

	if _, ok := validTriggerModes[triggerMode]; ok {
		return triggerMode, nil
	}

	return "", fmt.Errorf("invalid trigger mode: %s", triggerMode)
}

// Create creates a new task execution record
func (m *ExecutionModel) Create(ctx context.Context, taskID, agentID uuid.UUID, triggerMode string) (*TaskExecution, error) {
	triggerMode, err := NormalizeTriggerMode(triggerMode)
	if err != nil {
		return nil, err
	}

	execution := &TaskExecution{
		ID:          uuid.New(),
		TaskID:      taskID,
		AgentID:     agentID,
		Status:      "pending",
		TriggerMode: triggerMode,
		CreatedAt:   time.Now(),
	}

	query := `
		INSERT INTO task_executions (id, task_id, agent_id, status, trigger_mode, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	err = m.db.QueryRow(ctx, query,
		execution.ID,
		execution.TaskID,
		execution.AgentID,
		execution.Status,
		execution.TriggerMode,
		execution.CreatedAt,
	).Scan(&execution.ID, &execution.CreatedAt)

	if err != nil {
		return nil, err
	}

	return execution, nil
}

// GetByID retrieves an execution by ID
func (m *ExecutionModel) GetByID(ctx context.Context, id uuid.UUID) (*TaskExecution, error) {
	execution := &TaskExecution{}
	query := `
		SELECT 
			e.id, e.task_id, e.agent_id, e.status, e.trigger_mode,
			COALESCE(e.log_output, ''), COALESCE(e.error_message, ''), e.bytes_transferred, e.files_transferred,
			e.duration_seconds, e.started_at, e.ended_at, e.created_at,
			COALESCE(t.name, ''), COALESCE(a.name, '')
		FROM task_executions e
		LEFT JOIN backup_tasks t ON e.task_id = t.id
		LEFT JOIN agents a ON e.agent_id = a.id
		WHERE e.id = $1
	`

	err := m.db.QueryRow(ctx, query, id).Scan(
		&execution.ID,
		&execution.TaskID,
		&execution.AgentID,
		&execution.Status,
		&execution.TriggerMode,
		&execution.LogOutput,
		&execution.ErrorMessage,
		&execution.BytesTransferred,
		&execution.FilesTransferred,
		&execution.DurationSeconds,
		&execution.StartedAt,
		&execution.EndedAt,
		&execution.CreatedAt,
		&execution.TaskName,
		&execution.AgentName,
	)

	if err != nil {
		return nil, err
	}

	return execution, nil
}

// List retrieves executions with optional filters
func (m *ExecutionModel) List(ctx context.Context, limit int, offset int) ([]*TaskExecution, error) {
	query := `
		SELECT 
			e.id, e.task_id, e.agent_id, e.status, e.trigger_mode,
			e.bytes_transferred, e.files_transferred, e.duration_seconds,
			e.started_at, e.ended_at, e.created_at,
			t.name as task_name, a.name as agent_name
		FROM task_executions e
		LEFT JOIN backup_tasks t ON e.task_id = t.id
		LEFT JOIN agents a ON e.agent_id = a.id
		ORDER BY e.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := m.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	executions := []*TaskExecution{}
	for rows.Next() {
		execution := &TaskExecution{}
		err := rows.Scan(
			&execution.ID,
			&execution.TaskID,
			&execution.AgentID,
			&execution.Status,
			&execution.TriggerMode,
			&execution.BytesTransferred,
			&execution.FilesTransferred,
			&execution.DurationSeconds,
			&execution.StartedAt,
			&execution.EndedAt,
			&execution.CreatedAt,
			&execution.TaskName,
			&execution.AgentName,
		)
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}

	return executions, nil
}

// GetPendingForAgent gets pending executions for a specific agent
func (m *ExecutionModel) GetPendingForAgent(ctx context.Context, agentID uuid.UUID) ([]*TaskExecution, error) {
	query := `
		SELECT id, task_id, agent_id, status, trigger_mode, created_at
		FROM task_executions
		WHERE agent_id = $1 AND status = 'pending'
		ORDER BY created_at ASC
		LIMIT 10
	`

	rows, err := m.db.Query(ctx, query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	executions := []*TaskExecution{}
	for rows.Next() {
		execution := &TaskExecution{}
		err := rows.Scan(
			&execution.ID,
			&execution.TaskID,
			&execution.AgentID,
			&execution.Status,
			&execution.TriggerMode,
			&execution.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}

	return executions, nil
}

// GetCancelledForAgent gets executions in 'cancelled' status that need to be signaled
func (m *ExecutionModel) GetCancelledForAgent(ctx context.Context, agentID uuid.UUID) ([]*TaskExecution, error) {
	query := `
		SELECT id, task_id, agent_id, status, trigger_mode, created_at
		FROM task_executions
		WHERE agent_id = $1 AND status = 'cancelled'
		ORDER BY created_at ASC
		LIMIT 10
	`

	rows, err := m.db.Query(ctx, query, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	executions := []*TaskExecution{}
	for rows.Next() {
		execution := &TaskExecution{}
		err := rows.Scan(
			&execution.ID,
			&execution.TaskID,
			&execution.AgentID,
			&execution.Status,
			&execution.TriggerMode,
			&execution.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}

	return executions, nil
}

// UpdateStatus updates the status of an execution
func (m *ExecutionModel) UpdateStatus(ctx context.Context, id uuid.UUID, status string, endedAt *time.Time) error {
	now := time.Now()

	query := `
		UPDATE task_executions
		SET status = $2, started_at = COALESCE(started_at, $3)
	`

	args := []interface{}{id, status, now}

	if endedAt != nil {
		query += `, ended_at = $4 WHERE id = $1`
		args = append(args, *endedAt)
	} else {
		query += ` WHERE id = $1`
	}

	_, err := m.db.Exec(ctx, query, args...)
	return err
}

// UpdateLogs updates the log output of an execution (replaces existing logs)
func (m *ExecutionModel) UpdateLogs(ctx context.Context, id uuid.UUID, logs string) error {
	query := `
		UPDATE task_executions
		SET log_output = $2
		WHERE id = $1
	`

	_, err := m.db.Exec(ctx, query, id, logs)
	return err
}

// AppendLogs appends logs to existing execution log output
func (m *ExecutionModel) AppendLogs(ctx context.Context, id uuid.UUID, newLogs string) error {
	query := `
		UPDATE task_executions
		SET log_output = COALESCE(log_output, '') || $2
		WHERE id = $1
	`

	_, err := m.db.Exec(ctx, query, id, newLogs)
	return err
}

// UpdateErrorMessage updates the error message for an execution (replaces existing value).
func (m *ExecutionModel) UpdateErrorMessage(ctx context.Context, id uuid.UUID, message string) error {
	query := `
		UPDATE task_executions
		SET error_message = $2
		WHERE id = $1
	`

	_, err := m.db.Exec(ctx, query, id, strings.TrimSpace(message))
	return err
}

// StreamLogs adds timestamped log entries to execution
func (m *ExecutionModel) StreamLogs(ctx context.Context, id uuid.UUID, logEntries []LogEntry) error {
	var builder strings.Builder
	for _, entry := range logEntries {
		_, _ = fmt.Fprintf(&builder, "[%s] %s\n", entry.Timestamp, entry.Message)
	}

	if builder.Len() == 0 {
		return nil
	}
	return m.AppendLogs(ctx, id, builder.String())
}

// LogEntry represents a single log entry with timestamp
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// GetStatsByAgent gets execution statistics for an agent
func (m *ExecutionModel) GetStatsByAgent(ctx context.Context, agentID uuid.UUID, days int) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			AVG(EXTRACT(EPOCH FROM (ended_at - started_at))) as avg_duration
		FROM task_executions
		WHERE agent_id = $1 
		AND created_at > NOW() - INTERVAL '%d days'
	`

	var total, success, failed int
	var avgDuration *float64

	err := m.db.QueryRow(ctx, fmt.Sprintf(query, days), agentID).Scan(
		&total,
		&success,
		&failed,
		&avgDuration,
	)

	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total":        total,
		"success":      success,
		"failed":       failed,
		"avg_duration": avgDuration,
	}

	return stats, nil
}

// GetStatsByTask gets execution statistics for a task
func (m *ExecutionModel) GetStatsByTask(ctx context.Context, taskID uuid.UUID, days int) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'success' THEN 1 END) as success,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
			AVG(duration_seconds) as avg_duration
		FROM task_executions
		WHERE task_id = $1 
		AND created_at > NOW() - INTERVAL '%d days'
	`

	var total, success, failed int
	var avgDuration *float64

	err := m.db.QueryRow(ctx, fmt.Sprintf(query, days), taskID).Scan(
		&total,
		&success,
		&failed,
		&avgDuration,
	)

	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total":        total,
		"success":      success,
		"failed":       failed,
		"avg_duration": avgDuration,
	}

	return stats, nil
}

// GetByTimeRange gets executions within a time range
func (m *ExecutionModel) GetByTimeRange(ctx context.Context, days int) ([]*TaskExecution, error) {
	query := `
		SELECT 
			e.id, e.task_id, e.agent_id, e.status, e.trigger_mode,
			e.bytes_transferred, e.files_transferred, e.duration_seconds,
			e.started_at, e.ended_at, e.created_at
		FROM task_executions e
		WHERE e.created_at > NOW() - INTERVAL '%d days'
		ORDER BY e.created_at DESC
	`

	rows, err := m.db.Query(ctx, fmt.Sprintf(query, days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	executions := []*TaskExecution{}
	for rows.Next() {
		execution := &TaskExecution{}
		err := rows.Scan(
			&execution.ID,
			&execution.TaskID,
			&execution.AgentID,
			&execution.Status,
			&execution.TriggerMode,
			&execution.BytesTransferred,
			&execution.FilesTransferred,
			&execution.DurationSeconds,
			&execution.StartedAt,
			&execution.EndedAt,
			&execution.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}

	return executions, nil
}

// GetLastExecutionForTask gets the most recent execution for a specific task
func (m *ExecutionModel) GetLastExecutionForTask(ctx context.Context, taskID uuid.UUID) (*TaskExecution, error) {
	execution := &TaskExecution{}
	query := `
		SELECT 
			id, task_id, agent_id, status, trigger_mode,
			COALESCE(log_output, ''), COALESCE(error_message, ''), started_at, ended_at, created_at
		FROM task_executions
		WHERE task_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	err := m.db.QueryRow(ctx, query, taskID).Scan(
		&execution.ID,
		&execution.TaskID,
		&execution.AgentID,
		&execution.Status,
		&execution.TriggerMode,
		&execution.LogOutput,
		&execution.ErrorMessage,
		&execution.StartedAt,
		&execution.EndedAt,
		&execution.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No previous execution
		}
		return nil, err
	}

	return execution, nil
}

// GetLastSuccessfulExecutionForTask gets the most recent successful execution for a task
func (m *ExecutionModel) GetLastSuccessfulExecutionForTask(ctx context.Context, taskID uuid.UUID) (*TaskExecution, error) {
	execution := &TaskExecution{}
	query := `
		SELECT 
			id, task_id, agent_id, status, trigger_mode,
			COALESCE(log_output, ''), COALESCE(error_message, ''), started_at, ended_at, created_at
		FROM task_executions
		WHERE task_id = $1 AND status = 'success'
		ORDER BY created_at DESC
		LIMIT 1
	`

	err := m.db.QueryRow(ctx, query, taskID).Scan(
		&execution.ID,
		&execution.TaskID,
		&execution.AgentID,
		&execution.Status,
		&execution.TriggerMode,
		&execution.LogOutput,
		&execution.ErrorMessage,
		&execution.StartedAt,
		&execution.EndedAt,
		&execution.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No successful execution
		}
		return nil, err
	}

	return execution, nil
}
