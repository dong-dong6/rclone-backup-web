package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	"github.com/rclone-backup-web/shared/logger"
)

// Global logger instance
var log *logger.Logger

func init() {
	// Initialize logger
	logFile, err := os.OpenFile("/var/log/rclone-agent/agent.log", 
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logFile = os.Stdout
	}
	
	log = logger.New(logger.INFO, logFile)
	
	// Load message templates
	locale := os.Getenv("LOG_LOCALE")
	if locale == "" {
		locale = "zh" // Default to Chinese
	}
	
	// Try to load message files
	messagesDir := "/usr/share/rclone-agent/messages"
	if _, err := os.Stat(messagesDir); os.IsNotExist(err) {
		messagesDir = "./messages" // Fallback to local directory
	}
	
	zhFile := filepath.Join(messagesDir, "zh.json")
	enFile := filepath.Join(messagesDir, "en.json")
	
	// Load Chinese messages
	if err := log.LoadMessagesFromFile("zh", zhFile); err != nil {
		// If file not found, use embedded messages
		log.LoadMessages("zh", getEmbeddedChineseMessages())
	}
	
	// Load English messages
	if err := log.LoadMessagesFromFile("en", enFile); err != nil {
		log.LoadMessages("en", getEmbeddedEnglishMessages())
	}
	
	log.SetLocale(locale)
}

// getEmbeddedChineseMessages returns embedded Chinese messages
func getEmbeddedChineseMessages() map[string]string {
	return map[string]string{
		"AgentStarting":          "Agent正在启动...",
		"AgentStarted":           "Agent启动成功",
		"ConfigDirCreated":       "配置缓存目录已创建：{dir}",
		"ConfigDirCreateFailed":  "创建配置缓存目录失败：{error}",
		"CachedConfigLoaded":     "已加载缓存配置，包含 {task_count} 个任务",
		"CachedConfigLoadFailed": "加载缓存配置失败：{error}",
		"HeartbeatLoopStarted":   "心跳循环已启动，间隔：{interval}",
		"SendingHeartbeat":       "发送心跳，状态：{status}",
		"ProcessingAction":       "处理操作：{action}",
		"ExecutingTask":          "执行任务 {task_id}，触发模式：{trigger_mode}",
		"TaskCompleted":          "任务完成，状态：{status}，耗时：{duration_s}秒",
		"LocalSchedulerSetup":    "本地调度器设置完成",
		"AgentStopping":          "Agent正在停止...",
		"AgentStopped":           "Agent已停止",
	}
}

// getEmbeddedEnglishMessages returns embedded English messages
func getEmbeddedEnglishMessages() map[string]string {
	return map[string]string{
		"AgentStarting":          "Agent starting...",
		"AgentStarted":           "Agent started successfully",
		"ConfigDirCreated":       "Config cache directory created: {dir}",
		"ConfigDirCreateFailed":  "Failed to create config cache directory: {error}",
		"CachedConfigLoaded":     "Loaded cached config with {task_count} tasks",
		"CachedConfigLoadFailed": "Failed to load cached config: {error}",
		"HeartbeatLoopStarted":   "Heartbeat loop started, interval: {interval}",
		"SendingHeartbeat":       "Sending heartbeat, status: {status}",
		"ProcessingAction":       "Processing action: {action}",
		"ExecutingTask":          "Executing task {task_id}, trigger mode: {trigger_mode}",
		"TaskCompleted":          "Task completed, status: {status}, duration: {duration_s}s",
		"LocalSchedulerSetup":    "Local scheduler setup complete",
		"AgentStopping":          "Agent stopping...",
		"AgentStopped":           "Agent stopped",
	}
}

// Config and other structs remain the same...
type Config struct {
	HubURL            string
	AgentID           string
	APIKey            string
	HeartbeatInterval time.Duration
	ConfigCacheDir    string
	RcloneEndpoint    string
}

type Agent struct {
	config        *Config
	httpClient    *http.Client
	cron          *cron.Cron
	taskCache     map[string]*Task
	taskCacheMux  sync.RWMutex
	isRunningTask bool
	runningMux    sync.Mutex
	logger        *logger.ContextLogger // Context logger for this agent
}

type Task struct {
	TaskID          string   `json:"task_id"`
	RemoteID        string   `json:"remote_id"`
	SourcePath      string   `json:"source_path"`
	DestinationPath string   `json:"destination_path"`
	Schedule        string   `json:"schedule"`
	RcloneArgs      []string `json:"rclone_args"`
	RcloneConfigB64 string   `json:"rclone_config_b64,omitempty"`
}

// NewAgent creates a new agent instance
func NewAgent(config *Config) *Agent {
	// Create context logger with agent details
	contextLogger := log.WithDetails(map[string]interface{}{
		"agent_id": config.AgentID,
	})
	
	return &Agent{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cron:      cron.New(cron.WithSeconds()),
		taskCache: make(map[string]*Task),
		logger:    contextLogger,
	}
}

// Start starts the agent
func (a *Agent) Start() error {
	a.logger.Info("AgentStarting", nil)

	// Create config cache directory
	if err := os.MkdirAll(a.config.ConfigCacheDir, 0755); err != nil {
		a.logger.Error("ConfigDirCreateFailed", map[string]interface{}{
			"dir":   a.config.ConfigCacheDir,
			"error": err.Error(),
		})
		return fmt.Errorf("failed to create config cache directory: %w", err)
	}
	a.logger.Info("ConfigDirCreated", map[string]interface{}{
		"dir": a.config.ConfigCacheDir,
	})

	// Load cached configuration
	if err := a.loadCachedConfig(); err != nil {
		a.logger.Warn("CachedConfigLoadFailed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Start cron scheduler for local fallback
	a.setupLocalScheduler()
	a.cron.Start()
	a.logger.Info("LocalSchedulerSetup", nil)

	// Start heartbeat loop
	go a.heartbeatLoop()

	a.logger.Info("AgentStarted", nil)
	return nil
}

// heartbeatLoop sends periodic heartbeats to the hub
func (a *Agent) heartbeatLoop() {
	ticker := time.NewTicker(a.config.HeartbeatInterval)
	defer ticker.Stop()

	a.logger.Info("HeartbeatLoopStarted", map[string]interface{}{
		"interval": a.config.HeartbeatInterval.String(),
	})

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

	a.logger.Debug("SendingHeartbeat", map[string]interface{}{
		"status": status,
	})

	reqBody := map[string]string{"status": status}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", a.config.HubURL+"/api/v1/agent/heartbeat", bytes.NewReader(bodyBytes))
	if err != nil {
		a.logger.Error("HeartbeatFailed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Error("HubConnectionFailed", map[string]interface{}{
			"error":      err.Error(),
			"duration_s": 30,
		})
		a.executeLocalFallback()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.logger.Warn("HeartbeatFailed", map[string]interface{}{
			"status_code": resp.StatusCode,
		})
		a.executeLocalFallback()
		return
	}

	// Parse response and process actions
	var heartbeatResp struct {
		Actions []struct {
			Action      string          `json:"action"`
			ExecutionID string          `json:"execution_id,omitempty"`
			Task        json.RawMessage `json:"task,omitempty"`
		} `json:"actions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err != nil {
		a.logger.Error("HeartbeatFailed", map[string]interface{}{
			"error": "Failed to decode response",
		})
		return
	}

	a.logger.Info("HeartbeatSent", map[string]interface{}{
		"status":       status,
		"action_count": len(heartbeatResp.Actions),
	})

	// Process actions
	for _, action := range heartbeatResp.Actions {
		a.logger.Info("ProcessingAction", map[string]interface{}{
			"action": action.Action,
		})

		switch action.Action {
		case "SYNC_CONFIG":
			go a.syncConfig()
		case "EXECUTE_TASK":
			var task Task
			if err := json.Unmarshal(action.Task, &task); err != nil {
				a.logger.Error("TaskExecutionFailed", map[string]interface{}{
					"error": "Failed to unmarshal task",
				})
				continue
			}
			go a.executeTask(action.ExecutionID, &task, "central")
		}
	}
}

// syncConfig syncs configuration from the hub
func (a *Agent) syncConfig() {
	a.logger.Info("ConfigSyncing", nil)

	req, err := http.NewRequest("GET", a.config.HubURL+"/api/v1/agent/tasks", nil)
	if err != nil {
		a.logger.Error("ConfigSyncFailed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Error("ConfigSyncFailed", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		a.logger.Error("ConfigSyncFailed", map[string]interface{}{
			"status_code": resp.StatusCode,
		})
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
		a.logger.Error("ConfigSyncFailed", map[string]interface{}{
			"error": "Failed to decode response",
		})
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
	taskCount := len(a.taskCache)
	a.taskCacheMux.Unlock()

	// Save to disk
	if err := a.saveCachedConfig(); err != nil {
		a.logger.Error("CacheSaveFailed", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		a.logger.Info("CacheSaved", nil)
	}

	// Update local scheduler
	a.updateLocalScheduler()

	a.logger.Info("ConfigSynced", map[string]interface{}{
		"task_count": taskCount,
	})
}

// executeTask executes a backup task
func (a *Agent) executeTask(executionID string, task *Task, triggerMode string) {
	a.logger.Info("TaskExecutionStarted", map[string]interface{}{
		"task_id":      task.TaskID,
		"execution_id": executionID,
		"trigger_mode": triggerMode,
		"task_name":    task.TaskID, // In real implementation, get task name
	})

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
		a.logger.Error("RcloneConfigFailed", map[string]interface{}{
			"error": err.Error(),
		})
		a.reportExecutionResult(executionID, "failed", fmt.Sprintf("Config error: %v", err), startTime)
		return
	}
	defer os.Remove(configFile)

	a.logger.Info("RcloneConfigPrepared", map[string]interface{}{
		"config_file": configFile,
	})

	// Build and execute rclone command
	args := []string{
		"sync",
		task.SourcePath,
		fmt.Sprintf("remote:%s", task.DestinationPath),
		"--config", configFile,
		"--verbose",
	}
	args = append(args, task.RcloneArgs...)

	cmdStr := fmt.Sprintf("rclone %s", strings.Join(args, " "))
	a.logger.Info("RcloneCommandExecuting", map[string]interface{}{
		"command": cmdStr,
	})

	cmd := exec.Command("rclone", args...)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start log streaming
	a.logger.Info("LogStreamingStarted", map[string]interface{}{
		"execution_id": executionID,
	})

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
		a.logger.Error("TaskExecutionFailed", map[string]interface{}{
			"task_name": task.TaskID,
			"error":     err.Error(),
		})
	} else {
		duration := time.Since(startTime).Seconds()
		a.logger.Info("TaskExecutionCompleted", map[string]interface{}{
			"task_name":   task.TaskID,
			"status":      status,
			"duration_s":  duration,
		})
	}

	// Report result
	a.reportExecutionResult(executionID, status, finalOutput, startTime)
}

// Other methods remain similar with appropriate logging...

// Stop stops the agent gracefully
func (a *Agent) Stop() {
	a.logger.Info("AgentStopping", nil)
	a.cron.Stop()
	a.logger.Info("AgentStopped", nil)
}

func main() {
	// System startup log
	log.Info("SystemStartup", map[string]interface{}{
		"version": "2.0.0",
	})

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Warn("ConfigurationError", map[string]interface{}{
			"error": ".env file not found",
		})
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
		log.Fatal("ConfigurationError", map[string]interface{}{
			"error": "AGENT_ID and AGENT_API_KEY must be set",
		})
	}

	log.Info("ConfigurationLoaded", map[string]interface{}{
		"config_file": ".env",
		"hub_url":     config.HubURL,
	})

	// Create agent
	agent := NewAgent(config)

	// Start agent
	if err := agent.Start(); err != nil {
		log.Fatal("AgentStartFailed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("SystemShutdown", nil)

	// Stop agent
	agent.Stop()
}