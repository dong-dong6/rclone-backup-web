package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rclone-backup-web/hub/models"
	"github.com/robfig/cron/v3"
)

// TaskService handles task scheduling and execution logic
type TaskService struct {
	db *pgxpool.Pool
}

// NewTaskService creates a new task service
func NewTaskService(db *pgxpool.Pool) *TaskService {
	return &TaskService{db: db}
}

// FindPendingTaskForAgent finds a task that needs to be executed by the agent
func (s *TaskService) FindPendingTaskForAgent(ctx context.Context, agentID uuid.UUID) (*models.BackupTask, error) {
	// First, check if there are any manually triggered pending executions
	executionModel := models.NewExecutionModel(s.db)
	pendingExecutions, err := executionModel.GetPendingForAgent(ctx, agentID)
	if err == nil && len(pendingExecutions) > 0 {
		// Return the task for the first pending execution
		taskModel := models.NewTaskModel(s.db)
		return taskModel.GetByID(ctx, pendingExecutions[0].TaskID)
	}

	// Second, check for scheduled tasks that need to run
	task, err := s.findScheduledTaskForAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// findScheduledTaskForAgent checks if any scheduled task should run now
func (s *TaskService) findScheduledTaskForAgent(ctx context.Context, agentID uuid.UUID) (*models.BackupTask, error) {
	// Get all active tasks assigned to this agent
	taskModel := models.NewTaskModel(s.db)
	tasks, err := taskModel.GetAgentTasks(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent tasks: %w", err)
	}

	now := time.Now()
	
	for _, task := range tasks {
		if !task.IsActive || task.Schedule == "" {
			continue
		}

		// Check if this task should run based on its schedule
		if s.shouldTaskRunNow(ctx, task, now) {
			log.Printf("Task %s (%s) is due for execution on agent %s", 
				task.ID, task.Name, agentID)
			return task, nil
		}
	}

	return nil, nil // No task needs to run
}

// shouldTaskRunNow determines if a task should run based on its schedule
func (s *TaskService) shouldTaskRunNow(ctx context.Context, task *models.BackupTask, now time.Time) bool {
	// Parse cron expression
	schedule, err := cron.ParseStandard(task.Schedule)
	if err != nil {
		log.Printf("Failed to parse cron expression for task %s: %v", task.ID, err)
		return false
	}

	// Get the last execution time for this task
	executionModel := models.NewExecutionModel(s.db)
	lastExecution, err := executionModel.GetLastExecutionForTask(ctx, task.ID)
	
	var lastRunTime time.Time
	if err != nil || lastExecution == nil {
		// No previous execution, use a time 24 hours ago as reference
		lastRunTime = now.Add(-24 * time.Hour)
	} else {
		if lastExecution.StartedAt != nil {
			lastRunTime = *lastExecution.StartedAt
		} else {
			lastRunTime = lastExecution.CreatedAt
		}
	}

	// Calculate next run time after last execution
	nextRunTime := schedule.Next(lastRunTime)

	// Check if it's time to run (with 1-minute tolerance for heartbeat delays)
	shouldRun := now.After(nextRunTime) || now.Equal(nextRunTime)
	
	// Also check that we're not too far past the scheduled time (e.g., 1 hour)
	// This prevents running very old missed schedules
	if shouldRun && now.After(nextRunTime.Add(1*time.Hour)) {
		log.Printf("Skipping task %s - scheduled time %v is too far in the past", 
			task.ID, nextRunTime)
		return false
	}

	if shouldRun {
		log.Printf("Task %s should run - last run: %v, next scheduled: %v, now: %v",
			task.ID, lastRunTime, nextRunTime, now)
	}

	return shouldRun
}

// CreateExecution creates a new task execution record
func (s *TaskService) CreateExecution(ctx context.Context, taskID, agentID uuid.UUID, triggerMode string) (*models.TaskExecution, error) {
	executionModel := models.NewExecutionModel(s.db)
	execution, err := executionModel.Create(ctx, taskID, agentID, triggerMode)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	log.Printf("Created execution %s for task %s on agent %s (trigger: %s)",
		execution.ID, taskID, agentID, triggerMode)

	return execution, nil
}

// BuildTaskDetailsForAgent prepares task details for agent execution
func (s *TaskService) BuildTaskDetailsForAgent(ctx context.Context, task *models.BackupTask, executionID uuid.UUID, cryptoService *CryptoService) (map[string]interface{}, error) {
	// Get remote configuration
	remoteModel := models.NewRemoteModel(s.db)
	remote, err := remoteModel.GetByID(ctx, task.RcloneRemoteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote config: %w", err)
	}

	// Decrypt remote configuration
	decryptedConfig, err := cryptoService.Decrypt(remote.ConfigData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt remote config: %w", err)
	}

	// Build task details
	taskDetails := map[string]interface{}{
		"execution_id":      executionID.String(),
		"task_id":           task.ID.String(),
		"task_name":         task.Name,
		"source_path":       task.SourcePath,
		"destination_path":  task.DestinationPath,
		"rclone_config": map[string]interface{}{
			"name":   remote.Name,
			"config": decryptedConfig,
		},
	}

	// Add rclone arguments if present
	if task.RcloneArgs != nil {
		var args []string
		if err := task.RcloneArgs.UnmarshalJSON(&args); err == nil {
			taskDetails["rclone_args"] = args
		}
	}

	return taskDetails, nil
}