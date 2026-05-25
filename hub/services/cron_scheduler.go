package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rclone-backup-web/hub/models"
	"github.com/robfig/cron/v3"
)

// CronScheduler manages cron-based task scheduling
type CronScheduler struct {
	db             *pgxpool.Pool
	parser         cron.Parser
	executionCache map[uuid.UUID]time.Time // taskID -> last execution time
	cacheMux       sync.RWMutex
	minInterval    time.Duration // Minimum interval between executions
}

// NewCronScheduler creates a new cron scheduler
func NewCronScheduler(db *pgxpool.Pool) *CronScheduler {
	return &CronScheduler{
		db:             db,
		parser:         cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
		executionCache: make(map[uuid.UUID]time.Time),
		minInterval:    1 * time.Minute, // Prevent tasks from running more than once per minute
	}
}

// GetDueTasksForAgent returns tasks that are due for execution
func (s *CronScheduler) GetDueTasksForAgent(ctx context.Context, agentID uuid.UUID) ([]*models.BackupTask, error) {
	// Get all active tasks assigned to this agent
	taskModel := models.NewTaskModel(s.db)
	tasks, err := taskModel.GetAgentTasks(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent tasks: %w", err)
	}

	var dueTasks []*models.BackupTask
	now := time.Now()

	for _, task := range tasks {
		if !task.IsActive || task.Schedule == "" {
			continue
		}

		isDue, reason := s.isTaskDue(ctx, task, now)
		if isDue {
			log.Printf("[Scheduler] Task %s (%s) is due: %s",
				task.ID, task.Name, reason)
			dueTasks = append(dueTasks, task)
		}
	}

	return dueTasks, nil
}

// isTaskDue determines if a task should run based on its cron schedule
func (s *CronScheduler) isTaskDue(ctx context.Context, task *models.BackupTask, now time.Time) (bool, string) {
	// Parse cron expression
	schedule, err := s.parser.Parse(task.Schedule)
	if err != nil {
		log.Printf("[Scheduler] Failed to parse cron for task %s: %v", task.ID, err)
		return false, "invalid cron expression"
	}

	// Get last successful execution time
	lastExecTime, err := s.getLastExecutionTime(ctx, task.ID)
	if err != nil {
		log.Printf("[Scheduler] Error getting last execution for task %s: %v", task.ID, err)
	}

	// If never executed, use a reference point in the past
	if lastExecTime.IsZero() {
		// For new tasks, use creation time as reference
		lastExecTime = task.CreatedAt

		// If task was created very recently, don't run immediately
		if now.Sub(lastExecTime) < 1*time.Minute {
			return false, "task just created, waiting for first schedule"
		}
	}

	// Check minimum interval (prevent rapid re-execution)
	timeSinceLastExec := now.Sub(lastExecTime)
	if timeSinceLastExec < s.minInterval {
		return false, fmt.Sprintf("minimum interval not met (last: %v ago)", timeSinceLastExec)
	}

	// Calculate next scheduled time after last execution
	nextScheduledTime := schedule.Next(lastExecTime)

	// Check if we've passed the scheduled time
	if now.Before(nextScheduledTime) {
		return false, fmt.Sprintf("next run at %v", nextScheduledTime.Format(time.RFC3339))
	}

	// Check if we're too far past the scheduled time (missed window)
	// This prevents executing very old schedules when system comes back online
	missedWindow := 2 * time.Hour
	if now.After(nextScheduledTime.Add(missedWindow)) {
		// Check if there's a more recent schedule we should wait for instead
		nextNextTime := schedule.Next(nextScheduledTime)
		if now.Before(nextNextTime) {
			return false, fmt.Sprintf("missed window for %v, waiting for %v",
				nextScheduledTime.Format(time.RFC3339),
				nextNextTime.Format(time.RFC3339))
		}
	}

	// Task is due!
	return true, fmt.Sprintf("scheduled for %v (last run: %v)",
		nextScheduledTime.Format(time.RFC3339),
		lastExecTime.Format(time.RFC3339))
}

// getLastExecutionTime gets the last successful execution time for a task
func (s *CronScheduler) getLastExecutionTime(ctx context.Context, taskID uuid.UUID) (time.Time, error) {
	// Check cache first
	s.cacheMux.RLock()
	if cachedTime, exists := s.executionCache[taskID]; exists {
		s.cacheMux.RUnlock()
		return cachedTime, nil
	}
	s.cacheMux.RUnlock()

	// Query database for last successful execution
	executionModel := models.NewExecutionModel(s.db)
	lastExec, err := executionModel.GetLastExecutionForTask(ctx, taskID)
	if err != nil {
		return time.Time{}, err
	}

	if lastExec == nil {
		// No previous execution
		return time.Time{}, nil
	}

	// Use ended_at if available, otherwise started_at, otherwise created_at
	var execTime time.Time
	if lastExec.EndedAt != nil {
		execTime = *lastExec.EndedAt
	} else if lastExec.StartedAt != nil {
		execTime = *lastExec.StartedAt
	} else {
		execTime = lastExec.CreatedAt
	}

	// Update cache
	s.cacheMux.Lock()
	s.executionCache[taskID] = execTime
	s.cacheMux.Unlock()

	return execTime, nil
}

// MarkTaskExecuted updates the last execution time for a task
func (s *CronScheduler) MarkTaskExecuted(taskID uuid.UUID, execTime time.Time) {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()
	s.executionCache[taskID] = execTime
}

// RefreshCache clears the execution time cache
func (s *CronScheduler) RefreshCache() {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()
	s.executionCache = make(map[uuid.UUID]time.Time)
	log.Printf("[Scheduler] Cache refreshed")
}

// GetNextRunTime returns the next scheduled run time for a task
func (s *CronScheduler) GetNextRunTime(ctx context.Context, task *models.BackupTask) (time.Time, error) {
	if !task.IsActive || task.Schedule == "" {
		return time.Time{}, fmt.Errorf("task is not active or has no schedule")
	}

	// Parse cron expression
	schedule, err := s.parser.Parse(task.Schedule)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Get last execution time
	lastExecTime, err := s.getLastExecutionTime(ctx, task.ID)
	if err != nil {
		// If error, use current time as reference
		lastExecTime = time.Now()
	}

	if lastExecTime.IsZero() {
		lastExecTime = time.Now()
	}

	// Calculate next run
	return schedule.Next(lastExecTime), nil
}

// ValidateCronExpression validates a cron expression
func (s *CronScheduler) ValidateCronExpression(expr string) error {
	_, err := s.parser.Parse(expr)
	return err
}
