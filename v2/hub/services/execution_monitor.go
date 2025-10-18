package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rclone-backup-web/hub/models"
)

// ExecutionMonitor monitors running executions and handles timeouts
type ExecutionMonitor struct {
	db               *pgxpool.Pool
	checkInterval    time.Duration
	executionTimeout time.Duration
	agentTimeout     time.Duration
	running          bool
}

// NewExecutionMonitor creates a new execution monitor
func NewExecutionMonitor(db *pgxpool.Pool) *ExecutionMonitor {
	return &ExecutionMonitor{
		db:               db,
		checkInterval:    1 * time.Minute,    // Check every minute
		executionTimeout: 2 * time.Hour,      // Max execution time
		agentTimeout:     5 * time.Minute,    // Agent offline threshold
		running:          false,
	}
}

// Start starts the execution monitor
func (m *ExecutionMonitor) Start(ctx context.Context) {
	if m.running {
		return
	}
	
	m.running = true
	log.Println("[ExecutionMonitor] Starting execution monitor")
	
	go m.monitorLoop(ctx)
}

// Stop stops the execution monitor
func (m *ExecutionMonitor) Stop() {
	m.running = false
	log.Println("[ExecutionMonitor] Stopping execution monitor")
}

// monitorLoop is the main monitoring loop
func (m *ExecutionMonitor) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.running = false
			return
		case <-ticker.C:
			if !m.running {
				return
			}
			m.checkExecutions(ctx)
		}
	}
}

// checkExecutions checks all running executions for issues
func (m *ExecutionMonitor) checkExecutions(ctx context.Context) {
	// Check for timed-out executions
	m.checkTimedOutExecutions(ctx)
	
	// Check for orphaned executions (agent offline)
	m.checkOrphanedExecutions(ctx)
	
	// Clean up old pending executions
	m.cleanupOldPendingExecutions(ctx)
}

// checkTimedOutExecutions marks executions that have been running too long as failed
func (m *ExecutionMonitor) checkTimedOutExecutions(ctx context.Context) {
	executionModel := models.NewExecutionModel(m.db)
	
	// Find executions that have been running for too long
	cutoffTime := time.Now().Add(-m.executionTimeout)
	
	query := `
		UPDATE task_executions
		SET 
			status = 'failed',
			ended_at = NOW(),
			error_message = 'Execution timeout exceeded'
		WHERE 
			status = 'running'
			AND started_at < $1
		RETURNING id, task_id, agent_id
	`
	
	rows, err := m.db.Query(ctx, query, cutoffTime)
	if err != nil {
		log.Printf("[ExecutionMonitor] Error checking timed-out executions: %v", err)
		return
	}
	defer rows.Close()
	
	count := 0
	for rows.Next() {
		var execID, taskID, agentID uuid.UUID
		if err := rows.Scan(&execID, &taskID, &agentID); err != nil {
			continue
		}
		
		count++
		log.Printf("[ExecutionMonitor] Marked execution %s as failed due to timeout (task: %s, agent: %s)",
			execID, taskID, agentID)
	}
	
	if count > 0 {
		log.Printf("[ExecutionMonitor] Marked %d executions as failed due to timeout", count)
	}
}

// checkOrphanedExecutions marks executions for offline agents as failed
func (m *ExecutionMonitor) checkOrphanedExecutions(ctx context.Context) {
	// Find running executions where the agent hasn't sent a heartbeat recently
	offlineThreshold := time.Now().Add(-m.agentTimeout)
	
	query := `
		UPDATE task_executions te
		SET 
			status = 'failed',
			ended_at = NOW(),
			error_message = 'Agent went offline during execution'
		FROM agents a
		WHERE 
			te.agent_id = a.id
			AND te.status = 'running'
			AND (a.last_heartbeat < $1 OR a.status = 'offline')
		RETURNING te.id, te.task_id, te.agent_id
	`
	
	rows, err := m.db.Query(ctx, query, offlineThreshold)
	if err != nil {
		log.Printf("[ExecutionMonitor] Error checking orphaned executions: %v", err)
		return
	}
	defer rows.Close()
	
	count := 0
	for rows.Next() {
		var execID, taskID, agentID uuid.UUID
		if err := rows.Scan(&execID, &taskID, &agentID); err != nil {
			continue
		}
		
		count++
		log.Printf("[ExecutionMonitor] Marked execution %s as failed due to offline agent (task: %s, agent: %s)",
			execID, taskID, agentID)
	}
	
	if count > 0 {
		log.Printf("[ExecutionMonitor] Marked %d executions as failed due to offline agents", count)
	}
}

// cleanupOldPendingExecutions removes pending executions that are too old
func (m *ExecutionMonitor) cleanupOldPendingExecutions(ctx context.Context) {
	// Delete pending executions older than 1 hour
	cutoffTime := time.Now().Add(-1 * time.Hour)
	
	query := `
		DELETE FROM task_executions
		WHERE 
			status = 'pending'
			AND created_at < $1
		RETURNING id
	`
	
	rows, err := m.db.Query(ctx, query, cutoffTime)
	if err != nil {
		log.Printf("[ExecutionMonitor] Error cleaning up old pending executions: %v", err)
		return
	}
	defer rows.Close()
	
	count := 0
	for rows.Next() {
		var execID uuid.UUID
		if err := rows.Scan(&execID); err != nil {
			continue
		}
		count++
	}
	
	if count > 0 {
		log.Printf("[ExecutionMonitor] Cleaned up %d old pending executions", count)
	}
}

// HandleExecutionStart updates execution when it starts
func (m *ExecutionMonitor) HandleExecutionStart(ctx context.Context, executionID uuid.UUID) error {
	now := time.Now()
	executionModel := models.NewExecutionModel(m.db)
	return executionModel.UpdateStatus(ctx, executionID, "running", &now)
}

// HandleExecutionComplete updates execution when it completes
func (m *ExecutionMonitor) HandleExecutionComplete(ctx context.Context, executionID uuid.UUID, success bool, logOutput string) error {
	status := "success"
	if !success {
		status = "failed"
	}
	
	now := time.Now()
	executionModel := models.NewExecutionModel(m.db)
	
	// Update status
	if err := executionModel.UpdateStatus(ctx, executionID, status, &now); err != nil {
		return err
	}
	
	// Update logs if provided
	if logOutput != "" {
		if err := executionModel.UpdateLogs(ctx, executionID, logOutput); err != nil {
			log.Printf("[ExecutionMonitor] Failed to update logs for execution %s: %v", 
				executionID, err)
		}
	}
	
	return nil
}

// GetExecutionStats returns statistics about executions
func (m *ExecutionMonitor) GetExecutionStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'running') as running,
			COUNT(*) FILTER (WHERE status = 'success') as success,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COUNT(*) FILTER (WHERE status = 'success' AND ended_at > NOW() - INTERVAL '24 hours') as success_24h,
			COUNT(*) FILTER (WHERE status = 'failed' AND ended_at > NOW() - INTERVAL '24 hours') as failed_24h,
			AVG(EXTRACT(EPOCH FROM (ended_at - started_at))) FILTER (WHERE status = 'success') as avg_duration_seconds
		FROM task_executions
		WHERE created_at > NOW() - INTERVAL '7 days'
	`
	
	var stats struct {
		Pending            int
		Running            int
		Success            int
		Failed             int
		Success24h         int
		Failed24h          int
		AvgDurationSeconds *float64
	}
	
	err := m.db.QueryRow(ctx, query).Scan(
		&stats.Pending,
		&stats.Running,
		&stats.Success,
		&stats.Failed,
		&stats.Success24h,
		&stats.Failed24h,
		&stats.AvgDurationSeconds,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to get execution stats: %w", err)
	}
	
	result := map[string]interface{}{
		"pending":     stats.Pending,
		"running":     stats.Running,
		"success":     stats.Success,
		"failed":      stats.Failed,
		"success_24h": stats.Success24h,
		"failed_24h":  stats.Failed24h,
	}
	
	if stats.AvgDurationSeconds != nil {
		result["avg_duration_seconds"] = *stats.AvgDurationSeconds
	}
	
	// Calculate success rate
	total24h := stats.Success24h + stats.Failed24h
	if total24h > 0 {
		result["success_rate_24h"] = float64(stats.Success24h) / float64(total24h) * 100
	}
	
	return result, nil
}