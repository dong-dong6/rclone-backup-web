package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
)

const (
	triggerModeScheduled     = "scheduled"
	triggerModeManual        = "manual"
	triggerModeRetry         = "retry"
	triggerModeLocalFallback = "local_fallback"
)

var i18nLogMessages = map[string]map[string]string{
	"en": {
		"TaskExecutionStarted":   "Task {task_id} started (execution {execution_id}, trigger {trigger})",
		"TaskExecutionCompleted": "Task {task_id} finished with status {status}, duration {duration_ms} ms (execution {execution_id})",
		"TaskCancelled":          "Task {task_id} cancelled (execution {execution_id})",
		"ConfigSyncStarted":      "Syncing configuration from hub",
		"ConfigSynced":           "Configuration synced successfully, tasks: {task_count}",
		"ConfigSyncFailed":       "Configuration sync failed: {error}",
		"HeartbeatFailed":        "Heartbeat failed: {error}",
	},
	"zh": {
		"TaskExecutionStarted":   "开始执行任务 {task_id}（执行ID {execution_id}，触发方式 {trigger}）",
		"TaskExecutionCompleted": "任务 {task_id} 完成，状态 {status}，耗时 {duration_ms} 毫秒（执行ID {execution_id}）",
		"TaskCancelled":          "任务 {task_id} 已取消（执行ID {execution_id}）",
		"ConfigSyncStarted":      "开始从 Hub 同步配置",
		"ConfigSynced":           "配置同步完成，任务数 {task_count}",
		"ConfigSyncFailed":       "配置同步失败：{error}",
		"HeartbeatFailed":        "心跳发送失败：{error}",
	},
}

// Config holds agent configuration
type Config struct {
	HubURL            string
	AgentID           string
	APIKey            string
	HeartbeatInterval time.Duration
	ConfigCacheDir    string
	RcloneEndpoint    string
}

// Agent represents the backup agent
type Agent struct {
	config                  *Config
	httpClient              *http.Client
	cron                    *cron.Cron
	taskCache               map[string]*Task
	taskCacheMux            sync.RWMutex
	isRunningTask           bool
	runningMux              sync.Mutex
	runningTasks            map[string]context.CancelFunc
	runningTasksMux         sync.Mutex
	lastSuccessfulHeartbeat time.Time
	lastTaskExecution       map[string]time.Time
	hubReachable            bool
	logLocale               string
}

// Task represents a backup task
type Task struct {
	TaskID          string   `json:"task_id"`
	RemoteID        string   `json:"remote_id"`
	SourcePath      string   `json:"source_path"`
	DestinationPath string   `json:"destination_path"`
	Schedule        string   `json:"schedule"`
	RcloneArgs      []string `json:"rclone_args"`
	RcloneConfigB64 string   `json:"rclone_config_b64,omitempty"`
}

// HeartbeatRequest represents the heartbeat request payload
type HeartbeatRequest struct {
	Status string `json:"status"`
}

// HeartbeatResponse represents the heartbeat response
type HeartbeatResponse struct {
	Actions []HeartbeatAction `json:"actions"`
}

type HeartbeatAction struct {
	Action      string          `json:"action"`
	ExecutionID string          `json:"execution_id,omitempty"`
	TriggerMode string          `json:"trigger_mode,omitempty"`
	Task        json.RawMessage `json:"task,omitempty"`
}

func normalizeTriggerMode(mode string) string {
	switch mode {
	case triggerModeScheduled, triggerModeManual, triggerModeRetry, triggerModeLocalFallback:
		return mode
	default:
		return triggerModeScheduled
	}
}

func (a *Agent) logI18n(level, code string, details map[string]interface{}) {
	template := i18nLogMessages[a.logLocale][code]
	if template == "" {
		template = i18nLogMessages["en"][code]
	}
	if template == "" {
		template = code
	}

	message := template
	for key, value := range details {
		message = strings.ReplaceAll(message, "{"+key+"}", fmt.Sprintf("%v", value))
	}

	log.Printf("[%s] %s", strings.ToUpper(level), message)
}

// NewAgent creates a new agent instance
func NewAgent(config *Config) *Agent {
	locale := os.Getenv("LOG_LOCALE")
	if locale == "" {
		locale = "zh"
	}

	return &Agent{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cron:              cron.New(cron.WithSeconds()),
		taskCache:         make(map[string]*Task),
		runningTasks:      make(map[string]context.CancelFunc),
		lastTaskExecution: make(map[string]time.Time),
		hubReachable:      true,
		logLocale:         locale,
	}
}

// Start starts the agent
func (a *Agent) Start() error {
	log.Println("Starting agent...")

	// Create config cache directory
	if err := os.MkdirAll(a.config.ConfigCacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create config cache directory: %w", err)
	}

	// Load cached configuration
	if err := a.loadCachedConfig(); err != nil {
		log.Printf("Warning: Failed to load cached config: %v", err)
	}

	// Start cron scheduler for local fallback
	a.setupLocalScheduler()
	a.cron.Start()

	// Start heartbeat loop
	go a.heartbeatLoop()

	log.Println("Agent started successfully")
	return nil
}

// heartbeatLoop sends periodic heartbeats to the hub
func (a *Agent) heartbeatLoop() {
	ticker := time.NewTicker(a.config.HeartbeatInterval)
	defer ticker.Stop()

	// Send initial heartbeat immediately
	a.sendHeartbeat()

	for range ticker.C {
		a.sendHeartbeat()
	}
}

func (a *Agent) cancelTask(executionID string) {
	a.runningTasksMux.Lock()
	cancel, ok := a.runningTasks[executionID]
	a.runningTasksMux.Unlock()

	if !ok {
		log.Printf("No running task found for execution %s to cancel", executionID)
		return
	}

	log.Printf("Cancelling execution %s", executionID)
	cancel()
}

// sendHeartbeat sends a heartbeat to the hub
func (a *Agent) sendHeartbeat() {
	a.runningMux.Lock()
	status := "idle"
	if a.isRunningTask {
		status = "running_task"
	}
	a.runningMux.Unlock()

	reqBody := HeartbeatRequest{Status: status}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", a.config.HubURL+"/api/v1/agent/heartbeat", bytes.NewReader(bodyBytes))
	if err != nil {
		a.logI18n("error", "HeartbeatFailed", map[string]interface{}{"error": err})
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logI18n("error", "HeartbeatFailed", map[string]interface{}{"error": err})
		a.executeLocalFallback()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.logI18n("error", "HeartbeatFailed", map[string]interface{}{"error": resp.StatusCode})
		a.hubReachable = false
		a.executeLocalFallback()
		return
	}

	// Mark successful heartbeat
	a.runningMux.Lock()
	a.lastSuccessfulHeartbeat = time.Now()
	a.hubReachable = true
	a.runningMux.Unlock()

	var heartbeatResp HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err != nil {
		a.logI18n("error", "HeartbeatFailed", map[string]interface{}{"error": err})
		return
	}

	// Process actions
	for _, action := range heartbeatResp.Actions {
		switch action.Action {
		case "SYNC_CONFIG":
			go a.syncConfig()
		case "EXECUTE_TASK":
			var task Task
			if err := json.Unmarshal(action.Task, &task); err != nil {
				log.Printf("Failed to unmarshal task: %v", err)
				continue
			}
			go a.executeTask(action.ExecutionID, &task, normalizeTriggerMode(action.TriggerMode))
		case "CANCEL_TASK":
			a.cancelTask(action.ExecutionID)
		}
	}
}

// syncConfig syncs configuration from the hub
func (a *Agent) syncConfig() {
	a.logI18n("info", "ConfigSyncStarted", nil)

	req, err := http.NewRequest("GET", a.config.HubURL+"/api/v1/agent/tasks", nil)
	if err != nil {
		a.logI18n("error", "ConfigSyncFailed", map[string]interface{}{"error": err})
		return
	}

	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logI18n("error", "ConfigSyncFailed", map[string]interface{}{"error": err})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.logI18n("error", "ConfigSyncFailed", map[string]interface{}{"error": resp.StatusCode})
		return
	}

	var configResp struct {
		Tasks   []*Task `json:"tasks"`
		Remotes []struct {
			RemoteID  string `json:"remote_id"`
			ConfigB64 string `json:"config_b64"`
		} `json:"remotes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		a.logI18n("error", "ConfigSyncFailed", map[string]interface{}{"error": err})
		return
	}

	// Update task cache
	a.taskCacheMux.Lock()
	a.taskCache = make(map[string]*Task)
	for _, task := range configResp.Tasks {
		// Find matching remote config
		for _, remote := range configResp.Remotes {
			if remote.RemoteID == task.RemoteID {
				task.RcloneConfigB64 = remote.ConfigB64
				break
			}
		}
		a.taskCache[task.TaskID] = task
	}
	a.taskCacheMux.Unlock()

	// Save to disk
	if err := a.saveCachedConfig(); err != nil {
		a.logI18n("error", "ConfigSyncFailed", map[string]interface{}{"error": err})
	}

	// Update local scheduler
	a.updateLocalScheduler()

	a.logI18n("info", "ConfigSynced", map[string]interface{}{"task_count": len(configResp.Tasks)})
}

// executeTask executes a backup task
func (a *Agent) executeTask(executionID string, task *Task, triggerMode string) {
	a.logI18n("info", "TaskExecutionStarted", map[string]interface{}{
		"task_id":      task.TaskID,
		"execution_id": executionID,
		"trigger":      triggerMode,
	})

	ctx, cancel := context.WithCancel(context.Background())
	a.runningTasksMux.Lock()
	a.runningTasks[executionID] = cancel
	a.runningTasksMux.Unlock()

	a.runningMux.Lock()
	a.isRunningTask = true
	a.runningMux.Unlock()
	defer func() {
		a.runningMux.Lock()
		a.isRunningTask = false
		a.runningMux.Unlock()
		a.runningTasksMux.Lock()
		delete(a.runningTasks, executionID)
		a.runningTasksMux.Unlock()
	}()

	// For now, always use direct execution
	// TODO: Implement sidecar support
	a.executeTaskDirect(ctx, executionID, task, triggerMode)
}

// executeTaskDirect executes task directly (fallback method)
func (a *Agent) executeTaskDirect(ctx context.Context, executionID string, task *Task, triggerMode string) {
	startTime := time.Now()

	// Prepare rclone configuration
	configFile, err := a.prepareRcloneConfig(task)
	if err != nil {
		log.Printf("Failed to prepare rclone config: %v", err)
		a.reportExecutionResult(executionID, "failed", fmt.Sprintf("Config error: %v", err), startTime)
		return
	}
	defer os.Remove(configFile)

	// Build rclone command
	args := []string{
		"sync",
		task.SourcePath,
		fmt.Sprintf("remote:%s", task.DestinationPath),
		"--config", configFile,
		"--verbose",
	}
	args = append(args, task.RcloneArgs...)

	cmd := exec.CommandContext(ctx, "rclone", args...)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Stream logs periodically
	logTicker := time.NewTicker(5 * time.Second)
	defer logTicker.Stop()
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-logTicker.C:
				if stdout.Len() > 0 || stderr.Len() > 0 {
					a.streamLogs(executionID, stdout.String()+stderr.String())
				}
			}
		}
	}()

	// Execute command
	err = cmd.Run()
	close(done)

	// Final log stream
	finalOutput := stdout.String() + stderr.String()
	a.streamLogs(executionID, finalOutput)

	durationMs := time.Since(startTime).Milliseconds()
	status := "success"
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			status = "cancelled"
			finalOutput = fmt.Sprintf("Task cancelled\n\n%s", finalOutput)
			a.logI18n("warn", "TaskCancelled", map[string]interface{}{
				"task_id":      task.TaskID,
				"execution_id": executionID,
			})
		} else {
			status = "failed"
			finalOutput = fmt.Sprintf("Command failed: %v\n\n%s", err, finalOutput)
			a.logI18n("error", "TaskExecutionCompleted", map[string]interface{}{
				"task_id":      task.TaskID,
				"execution_id": executionID,
				"status":       status,
				"duration_ms":  durationMs,
			})
		}
	} else {
		a.logI18n("info", "TaskExecutionCompleted", map[string]interface{}{
			"task_id":      task.TaskID,
			"execution_id": executionID,
			"status":       status,
			"duration_ms":  durationMs,
		})
	}

	// Report result
	a.reportExecutionResult(executionID, status, finalOutput, startTime)
}

// prepareRcloneConfig prepares rclone configuration file
func (a *Agent) prepareRcloneConfig(task *Task) (string, error) {
	if task.RcloneConfigB64 == "" {
		return "", fmt.Errorf("no rclone config provided")
	}

	configData, err := base64.StdEncoding.DecodeString(task.RcloneConfigB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode config: %w", err)
	}

	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "rclone-*.conf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	// Write config with [remote] header if not present
	configStr := string(configData)
	if !strings.Contains(configStr, "[remote]") {
		configStr = "[remote]\n" + configStr
	}

	if _, err := tmpFile.WriteString(configStr); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write config: %w", err)
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}

// reportExecutionResult reports execution result to hub
func (a *Agent) reportExecutionResult(executionID, status, logOutput string, startTime time.Time) {
	endTime := time.Now()

	reqBody := map[string]interface{}{
		"status":     status,
		"log_output": logOutput,
		"ended_at":   endTime.Format(time.RFC3339),
	}

	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("PUT",
		fmt.Sprintf("%s/api/v1/agent/executions/%s", a.config.HubURL, executionID),
		bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("Failed to create execution report request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to report execution result: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to report execution result, status: %d", resp.StatusCode)
	}
}

// streamLogs streams logs to hub
func (a *Agent) streamLogs(executionID, logs string) {
	if logs == "" {
		return
	}

	lines := strings.Split(logs, "\n")
	logEntries := make([]map[string]string, 0, len(lines))

	for _, line := range lines {
		if line != "" {
			logEntries = append(logEntries, map[string]string{
				"timestamp": time.Now().Format(time.RFC3339),
				"message":   line,
			})
		}
	}

	if len(logEntries) == 0 {
		return
	}

	reqBody := map[string]interface{}{
		"logs": logEntries,
	}

	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v1/agent/executions/%s/logs", a.config.HubURL, executionID),
		bytes.NewReader(bodyBytes))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// setupLocalScheduler sets up local cron scheduler for fallback
func (a *Agent) setupLocalScheduler() {
	log.Println("Setting up local scheduler for fallback...")
}

// updateLocalScheduler updates local scheduler based on cached tasks
func (a *Agent) updateLocalScheduler() {
	// Clear existing jobs
	for _, entry := range a.cron.Entries() {
		a.cron.Remove(entry.ID)
	}

	a.taskCacheMux.RLock()
	defer a.taskCacheMux.RUnlock()

	for taskID, task := range a.taskCache {
		if task.Schedule == "" {
			continue
		}

		taskCopy := *task
		_, err := a.cron.AddFunc(task.Schedule, func() {
			// Only execute if we haven't heard from hub recently
			if a.shouldExecuteLocalFallback() {
				executionID := uuid.New().String()
				go a.executeTask(executionID, &taskCopy, triggerModeLocalFallback)
			}
		})

		if err != nil {
			log.Printf("Failed to schedule task %s: %v", taskID, err)
		} else {
			log.Printf("Scheduled task %s with cron: %s", taskID, task.Schedule)
		}
	}
}

// shouldExecuteLocalFallback checks if local fallback should be executed
func (a *Agent) shouldExecuteLocalFallback() bool {
	// Check if we have a last successful heartbeat
	a.runningMux.Lock()
	defer a.runningMux.Unlock()

	// If we haven't had a successful heartbeat in 5 minutes, enable local fallback
	if a.lastSuccessfulHeartbeat.IsZero() {
		return false // Never had a successful connection
	}

	timeSinceLastHeartbeat := time.Since(a.lastSuccessfulHeartbeat)
	return timeSinceLastHeartbeat > 5*time.Minute
}

// executeLocalFallback executes tasks based on local cache
func (a *Agent) executeLocalFallback() {
	log.Printf("Executing local fallback mode - Hub unreachable for %v", time.Since(a.lastSuccessfulHeartbeat))

	a.taskCacheMux.RLock()
	defer a.taskCacheMux.RUnlock()

	// Check each cached task
	for taskID, task := range a.taskCache {
		// Parse cron schedule
		schedule, err := cron.ParseStandard(task.Schedule)
		if err != nil {
			log.Printf("Failed to parse schedule for task %s: %v", taskID, err)
			continue
		}

		// Check if task should run now
		now := time.Now()
		next := schedule.Next(a.lastTaskExecution[taskID])

		if now.After(next) || now.Equal(next) {
			// Task should run
			log.Printf("Local fallback: Triggering task %s", taskID)

			// Generate a local execution ID
			executionID := fmt.Sprintf("local-%s-%d", taskID, now.Unix())

			// Execute task asynchronously
			go a.executeTask(executionID, task, triggerModeLocalFallback)

			// Update last execution time
			a.lastTaskExecution[taskID] = now
		}
	}
}

// loadCachedConfig loads configuration from disk
func (a *Agent) loadCachedConfig() error {
	configFile := filepath.Join(a.config.ConfigCacheDir, "tasks.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}

	var tasks []*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return err
	}

	a.taskCacheMux.Lock()
	a.taskCache = make(map[string]*Task)
	for _, task := range tasks {
		a.taskCache[task.TaskID] = task
	}
	a.taskCacheMux.Unlock()

	return nil
}

// saveCachedConfig saves configuration to disk
func (a *Agent) saveCachedConfig() error {
	a.taskCacheMux.RLock()
	tasks := make([]*Task, 0, len(a.taskCache))
	for _, task := range a.taskCache {
		tasks = append(tasks, task)
	}
	a.taskCacheMux.RUnlock()

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	configFile := filepath.Join(a.config.ConfigCacheDir, "tasks.json")
	return os.WriteFile(configFile, data, 0600)
}

// Stop stops the agent gracefully
func (a *Agent) Stop() {
	log.Println("Stopping agent...")
	a.cron.Stop()
	log.Println("Agent stopped")
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Load configuration
	config := &Config{
		HubURL:            getEnv("HUB_URL", "http://localhost:8080"),
		AgentID:           getEnv("AGENT_ID", ""),
		APIKey:            getEnv("AGENT_API_KEY", ""),
		HeartbeatInterval: getDurationEnv("HEARTBEAT_INTERVAL", 30*time.Second),
		ConfigCacheDir:    getEnv("CONFIG_CACHE_DIR", "/var/lib/rclone-agent"),
		RcloneEndpoint:    getEnv("RCLONE_ENDPOINT", "http://localhost:5572"),
	}

	if config.AgentID == "" || config.APIKey == "" {
		log.Fatal("AGENT_ID and AGENT_API_KEY must be set. Register agent first.")
	}

	// Create agent
	agent := NewAgent(config)

	// Start agent
	if err := agent.Start(); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// Stop agent
	agent.Stop()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
