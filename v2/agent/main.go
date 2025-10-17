package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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

// Config holds agent configuration
type Config struct {
	HubURL         string
	AgentID        string
	APIKey         string
	HeartbeatInterval time.Duration
	ConfigCacheDir string
	RcloneEndpoint string
}

// Agent represents the backup agent
type Agent struct {
	config        *Config
	httpClient    *http.Client
	cron          *cron.Cron
	taskCache     map[string]*Task
	taskCacheMux  sync.RWMutex
	isRunningTask bool
	runningMux    sync.Mutex
}

// Task represents a backup task
type Task struct {
	TaskID          string          `json:"task_id"`
	RemoteID        string          `json:"remote_id"`
	SourcePath      string          `json:"source_path"`
	DestinationPath string          `json:"destination_path"`
	Schedule        string          `json:"schedule"`
	RcloneArgs      []string        `json:"rclone_args"`
	RcloneConfigB64 string          `json:"rclone_config_b64,omitempty"`
}

// HeartbeatRequest represents the heartbeat request payload
type HeartbeatRequest struct {
	Status string `json:"status"`
}

// HeartbeatResponse represents the heartbeat response
type HeartbeatResponse struct {
	Actions []struct {
		Action      string          `json:"action"`
		ExecutionID string          `json:"execution_id,omitempty"`
		Task        json.RawMessage `json:"task,omitempty"`
	} `json:"actions"`
}

// NewAgent creates a new agent instance
func NewAgent(config *Config) *Agent {
	return &Agent{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cron:      cron.New(cron.WithSeconds()),
		taskCache: make(map[string]*Task),
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
		log.Printf("Failed to create heartbeat request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to send heartbeat: %v", err)
		a.executeLocalFallback()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Heartbeat failed with status: %d", resp.StatusCode)
		a.executeLocalFallback()
		return
	}

	var heartbeatResp HeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err != nil {
		log.Printf("Failed to decode heartbeat response: %v", err)
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
			go a.executeTask(action.ExecutionID, &task, "central")
		}
	}
}

// syncConfig syncs configuration from the hub
func (a *Agent) syncConfig() {
	log.Println("Syncing configuration from hub...")

	req, err := http.NewRequest("GET", a.config.HubURL+"/api/v1/agent/tasks", nil)
	if err != nil {
		log.Printf("Failed to create sync request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to sync config: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Config sync failed with status: %d", resp.StatusCode)
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
		log.Printf("Failed to decode config response: %v", err)
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
		log.Printf("Failed to save cached config: %v", err)
	}

	// Update local scheduler
	a.updateLocalScheduler()

	log.Println("Configuration synced successfully")
}

// executeTask executes a backup task
func (a *Agent) executeTask(executionID string, task *Task, triggerMode string) {
	log.Printf("Executing task %s (execution: %s, trigger: %s)", task.TaskID, executionID, triggerMode)

	a.runningMux.Lock()
	a.isRunningTask = true
	a.runningMux.Unlock()
	defer func() {
		a.runningMux.Lock()
		a.isRunningTask = false
		a.runningMux.Unlock()
	}()

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

	cmd := exec.Command("rclone", args...)
	
	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Stream logs periodically
	logTicker := time.NewTicker(5 * time.Second)
	defer logTicker.Stop()
	
	go func() {
		for range logTicker.C {
			if stdout.Len() > 0 || stderr.Len() > 0 {
				a.streamLogs(executionID, stdout.String()+stderr.String())
			}
		}
	}()

	// Execute command
	err = cmd.Run()
	
	// Final log stream
	finalOutput := stdout.String() + stderr.String()
	a.streamLogs(executionID, finalOutput)

	status := "success"
	if err != nil {
		status = "failed"
		finalOutput = fmt.Sprintf("Command failed: %v\n\n%s", err, finalOutput)
		log.Printf("Task execution failed: %v", err)
	} else {
		log.Printf("Task executed successfully")
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
				go a.executeTask(executionID, &taskCopy, "local_fallback")
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
	// Implement logic to check if hub is unreachable
	// For now, return false to prevent automatic execution
	return false
}

// executeLocalFallback executes tasks based on local cache
func (a *Agent) executeLocalFallback() {
	// This would be called when hub is unreachable
	// Implementation depends on specific requirements
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