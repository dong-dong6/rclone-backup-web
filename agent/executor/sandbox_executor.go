package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	
	"github.com/rclone-backup-web/agent/rclone_manager"
)

// TaskExecutor handles isolated task execution
type TaskExecutor struct {
	rcloneManager  *rclone_manager.Manager
	configManager  *rclone_manager.ConfigManager
	workDir        string
	maxConcurrent  int
	activeTasks    sync.Map
	taskSemaphore  chan struct{}
}

// TaskInfo contains task execution information
type TaskInfo struct {
	ID           string                 `json:"id"`
	ExecutionID  string                 `json:"execution_id"`
	TaskID       string                 `json:"task_id"`
	RemoteConfig string                 `json:"remote_config"`
	SourcePath   string                 `json:"source_path"`
	DestPath     string                 `json:"dest_path"`
	RcloneArgs   []string               `json:"rclone_args"`
	StartedAt    time.Time              `json:"started_at"`
	Status       string                 `json:"status"`
	Progress     *TransferProgress      `json:"progress,omitempty"`
	Logs         []string               `json:"-"` // Don't include in JSON
	Context      context.Context        `json:"-"`
	Cancel       context.CancelFunc     `json:"-"`
}

// TransferProgress tracks transfer statistics
type TransferProgress struct {
	BytesTransferred int64     `json:"bytes_transferred"`
	BytesTotal       int64     `json:"bytes_total"`
	FilesTransferred int       `json:"files_transferred"`
	FilesTotal       int       `json:"files_total"`
	Speed            string    `json:"speed"`
	Percentage       float64   `json:"percentage"`
	ETA              string    `json:"eta"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(workDir string, maxConcurrent int) (*TaskExecutor, error) {
	// Initialize rclone manager
	rcloneManager := rclone_manager.NewManager(workDir)
	if _, err := rcloneManager.EnsureRcloneExists(); err != nil {
		return nil, fmt.Errorf("failed to ensure rclone: %w", err)
	}
	
	// Initialize config manager
	configManager := rclone_manager.NewConfigManager(workDir)
	
	return &TaskExecutor{
		rcloneManager:  rcloneManager,
		configManager:  configManager,
		workDir:        workDir,
		maxConcurrent:  maxConcurrent,
		taskSemaphore:  make(chan struct{}, maxConcurrent),
	}, nil
}

// ExecuteTask executes a task in an isolated environment
func (te *TaskExecutor) ExecuteTask(ctx context.Context, task *TaskInfo) error {
	// Acquire semaphore to limit concurrent tasks
	select {
	case te.taskSemaphore <- struct{}{}:
		defer func() { <-te.taskSemaphore }()
	case <-ctx.Done():
		return ctx.Err()
	}
	
	// Create task context with cancellation
	taskCtx, cancel := context.WithCancel(ctx)
	task.Context = taskCtx
	task.Cancel = cancel
	defer cancel()
	
	// Store active task
	te.activeTasks.Store(task.ExecutionID, task)
	defer te.activeTasks.Delete(task.ExecutionID)
	
	// Update status
	task.Status = "preparing"
	task.StartedAt = time.Now()
	
	log.Printf("[Executor] Starting task %s (execution: %s)", task.TaskID, task.ExecutionID)
	
	// Create isolated work directory
	taskWorkDir := filepath.Join(te.workDir, "tasks", task.ExecutionID)
	if err := os.MkdirAll(taskWorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}
	defer te.cleanupTaskWorkDir(taskWorkDir)
	
	// Create temporary rclone config
	configPath, err := te.configManager.CreateTempConfig(task.ExecutionID, task.RemoteConfig)
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}
	defer te.configManager.CleanupConfig(task.ExecutionID)
	
	// Update status
	task.Status = "running"
	
	// Execute rclone command
	if err := te.runRclone(taskCtx, task, configPath, taskWorkDir); err != nil {
		task.Status = "failed"
		return fmt.Errorf("rclone execution failed: %w", err)
	}
	
	task.Status = "completed"
	log.Printf("[Executor] Task %s completed successfully", task.ExecutionID)
	
	return nil
}

// runRclone executes the rclone command
func (te *TaskExecutor) runRclone(ctx context.Context, task *TaskInfo, configPath, workDir string) error {
	// Build rclone command
	rclonePath := te.rcloneManager.GetExecutablePath()
	
	// Base command: rclone sync/copy source dest
	args := []string{
		"sync", // Default to sync, could be made configurable
		task.SourcePath,
		task.DestPath,
		"--config", configPath,
		"--progress",
		"--stats", "1s",
		"--stats-one-line",
		"--log-level", "INFO",
		"--use-json-log",
	}
	
	// Add user-specified arguments
	args = append(args, task.RcloneArgs...)
	
	log.Printf("[Executor] Running command: %s %s", rclonePath, strings.Join(args, " "))
	
	// Create command
	cmd := exec.CommandContext(ctx, rclonePath, args...)
	cmd.Dir = workDir
	
	// Set environment variables for isolation
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("RCLONE_CONFIG=%s", configPath),
		"RCLONE_NO_CHECK_CERTIFICATE=false",
		fmt.Sprintf("TMPDIR=%s", workDir),
	)
	
	// Capture stdout for progress parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	// Capture stderr for error messages
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start rclone: %w", err)
	}
	
	// Process output in goroutines
	var wg sync.WaitGroup
	wg.Add(2)
	
	// Process stdout (progress)
	go func() {
		defer wg.Done()
		te.processRcloneOutput(task, stdout, false)
	}()
	
	// Process stderr (logs)
	go func() {
		defer wg.Done()
		te.processRcloneOutput(task, stderr, true)
	}()
	
	// Wait for output processing
	wg.Wait()
	
	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("task cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("rclone command failed: %w", err)
	}
	
	return nil
}

// processRcloneOutput processes rclone output for progress and logs
func (te *TaskExecutor) processRcloneOutput(task *TaskInfo, reader io.Reader, isError bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		
		// Store log line
		task.Logs = append(task.Logs, line)
		
		// Try to parse as JSON log
		var jsonLog map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonLog); err == nil {
			te.parseRcloneJSON(task, jsonLog)
		} else {
			// Parse plain text progress
			te.parseRclonePlainText(task, line)
		}
		
		// Log to console
		if isError {
			log.Printf("[Executor][ERROR] %s", line)
		} else {
			log.Printf("[Executor] %s", line)
		}
	}
}

// parseRcloneJSON parses JSON formatted rclone output
func (te *TaskExecutor) parseRcloneJSON(task *TaskInfo, jsonLog map[string]interface{}) {
	// Look for stats in JSON
	if stats, ok := jsonLog["stats"].(map[string]interface{}); ok {
		progress := &TransferProgress{
			UpdatedAt: time.Now(),
		}
		
		if bytes, ok := stats["bytes"].(float64); ok {
			progress.BytesTransferred = int64(bytes)
		}
		if totalBytes, ok := stats["totalBytes"].(float64); ok {
			progress.BytesTotal = int64(totalBytes)
		}
		if transfers, ok := stats["transfers"].(float64); ok {
			progress.FilesTransferred = int(transfers)
		}
		if totalTransfers, ok := stats["totalTransfers"].(float64); ok {
			progress.FilesTotal = int(totalTransfers)
		}
		if speed, ok := stats["speed"].(float64); ok {
			progress.Speed = formatSpeed(int64(speed))
		}
		if eta, ok := stats["eta"].(float64); ok {
			progress.ETA = formatDuration(time.Duration(eta) * time.Second)
		}
		
		// Calculate percentage
		if progress.BytesTotal > 0 {
			progress.Percentage = float64(progress.BytesTransferred) / float64(progress.BytesTotal) * 100
		}
		
		task.Progress = progress
	}
}

// parseRclonePlainText parses plain text rclone output
func (te *TaskExecutor) parseRclonePlainText(task *TaskInfo, line string) {
	// Parse stats line like: "Transferred: 1.234 GiB / 5.678 GiB, 22%, 1.234 MiB/s, ETA 1h2m3s"
	if strings.Contains(line, "Transferred:") && strings.Contains(line, "%") {
		// This is a simplified parser - could be enhanced
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			// Extract percentage
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasSuffix(part, "%") {
					if task.Progress == nil {
						task.Progress = &TransferProgress{}
					}
					fmt.Sscanf(part, "%f%%", &task.Progress.Percentage)
					task.Progress.UpdatedAt = time.Now()
				}
			}
		}
	}
}

// CancelTask cancels a running task
func (te *TaskExecutor) CancelTask(executionID string) error {
	if val, ok := te.activeTasks.Load(executionID); ok {
		task := val.(*TaskInfo)
		if task.Cancel != nil {
			task.Cancel()
			task.Status = "cancelled"
			log.Printf("[Executor] Task %s cancelled", executionID)
			return nil
		}
	}
	return fmt.Errorf("task %s not found or not running", executionID)
}

// GetTaskStatus returns the status of a task
func (te *TaskExecutor) GetTaskStatus(executionID string) (*TaskInfo, bool) {
	if val, ok := te.activeTasks.Load(executionID); ok {
		return val.(*TaskInfo), true
	}
	return nil, false
}

// GetActiveTasks returns all active tasks
func (te *TaskExecutor) GetActiveTasks() []*TaskInfo {
	var tasks []*TaskInfo
	te.activeTasks.Range(func(key, value interface{}) bool {
		tasks = append(tasks, value.(*TaskInfo))
		return true
	})
	return tasks
}

// cleanupTaskWorkDir removes the task work directory
func (te *TaskExecutor) cleanupTaskWorkDir(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("[Executor] Warning: failed to cleanup work directory %s: %v", dir, err)
	}
}

// formatSpeed formats bytes per second as human readable
func formatSpeed(bps int64) string {
	const unit = 1024
	if bps < unit {
		return fmt.Sprintf("%d B/s", bps)
	}
	div, exp := int64(unit), 0
	for n := bps / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB/s", float64(bps)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration as human readable
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm%ds", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}