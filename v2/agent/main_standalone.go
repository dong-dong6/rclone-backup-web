package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rclone-backup-web/agent/executor"
	"github.com/rclone-backup-web/agent/services"
)

// Version information (set during build)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// Config holds agent configuration
type Config struct {
	// Hub connection
	HubURL           string `json:"hub_url"`
	RegistrationToken string `json:"registration_token,omitempty"`
	AgentID          string `json:"agent_id,omitempty"`
	APIKey           string `json:"api_key,omitempty"`
	
	// Agent settings
	AgentName        string `json:"agent_name"`
	WorkDir          string `json:"work_dir"`
	MaxConcurrent    int    `json:"max_concurrent"`
	HeartbeatInterval int   `json:"heartbeat_interval"`
	
	// Features
	EnableLocalFallback bool `json:"enable_local_fallback"`
	EnableAutoUpdate    bool `json:"enable_auto_update"`
	EnableMetrics       bool `json:"enable_metrics"`
	MetricsPort         int  `json:"metrics_port"`
	
	// System integration
	RunAsService     bool   `json:"run_as_service"`
	LogFile          string `json:"log_file"`
	PidFile          string `json:"pid_file"`
}

// Agent represents the standalone agent
type Agent struct {
	config       *Config
	executor     *executor.TaskExecutor
	hubClient    *services.HubClient
	scheduler    *services.Scheduler
	ctx          context.Context
	cancel       context.CancelFunc
}

func main() {
	// Parse command line flags
	var (
		configFile   = flag.String("config", "agent.json", "Configuration file path")
		showVersion  = flag.Bool("version", false, "Show version information")
		install      = flag.Bool("install", false, "Install as system service")
		uninstall    = flag.Bool("uninstall", false, "Uninstall system service")
		start        = flag.Bool("start", false, "Start system service")
		stop         = flag.Bool("stop", false, "Stop system service")
		workDir      = flag.String("work-dir", "", "Override work directory")
		hubURL       = flag.String("hub-url", "", "Override hub URL")
		token        = flag.String("token", "", "Registration token for first run")
		agentName    = flag.String("name", "", "Agent name (defaults to hostname)")
	)
	flag.Parse()
	
	// Show version
	if *showVersion {
		printVersion()
		return
	}
	
	// Handle service management commands
	if *install || *uninstall || *start || *stop {
		handleServiceCommand(*install, *uninstall, *start, *stop)
		return
	}
	
	// Load configuration
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Apply command line overrides
	if *workDir != "" {
		config.WorkDir = *workDir
	}
	if *hubURL != "" {
		config.HubURL = *hubURL
	}
	if *token != "" {
		config.RegistrationToken = *token
	}
	if *agentName != "" {
		config.AgentName = *agentName
	}
	
	// Setup logging
	if err := setupLogging(config); err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}
	
	log.Printf("Starting Rclone Backup Agent %s (standalone mode)", Version)
	log.Printf("Build: %s, Commit: %s", BuildTime, GitCommit)
	log.Printf("Runtime: %s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	
	// Create and run agent
	agent, err := NewAgent(config)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	
	// Write PID file
	if err := writePIDFile(config.PidFile); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}
	defer removePIDFile(config.PidFile)
	
	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Run agent
	go agent.Run()
	
	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal: %v", sig)
	
	// Graceful shutdown
	agent.Shutdown()
	log.Println("Agent shutdown complete")
}

// NewAgent creates a new standalone agent
func NewAgent(config *Config) (*Agent, error) {
	// Create work directory
	if err := os.MkdirAll(config.WorkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create work directory: %w", err)
	}
	
	// Create task executor
	taskExecutor, err := executor.NewTaskExecutor(config.WorkDir, config.MaxConcurrent)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}
	
	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	
	agent := &Agent{
		config:    config,
		executor:  taskExecutor,
		ctx:       ctx,
		cancel:    cancel,
		hubClient: services.NewHubClient(config.HubURL, "", ""),
	}
	
	// Register with hub if not already registered
	if err := agent.ensureRegistered(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to register: %w", err)
	}
	
	// Create scheduler for local tasks
	if config.EnableLocalFallback {
		agent.scheduler = services.NewScheduler(config.WorkDir)
	}
	
	return agent, nil
}

// Run starts the agent main loop
func (a *Agent) Run() {
	log.Println("Agent starting main loop")
	
	// Start metrics server if enabled
	if a.config.EnableMetrics {
		go a.startMetricsServer()
	}
	
	// Start local scheduler if enabled
	if a.scheduler != nil {
		go a.scheduler.Start(a.ctx)
	}
	
	// Main heartbeat loop
	ticker := time.NewTicker(time.Duration(a.config.HeartbeatInterval) * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.sendHeartbeat()
		}
	}
}

// sendHeartbeat sends heartbeat to hub and processes responses
func (a *Agent) sendHeartbeat() {
	// Get current status
	activeTasks := a.executor.GetActiveTasks()
	status := "online"
	if len(activeTasks) > 0 {
		status = "running_task"
	}
	
	// Send heartbeat
	response, err := a.hubClient.SendHeartbeat(a.ctx, status, activeTasks)
	if err != nil {
		log.Printf("Failed to send heartbeat: %v", err)
		
		// Handle local fallback
		if a.config.EnableLocalFallback {
			a.handleLocalFallback()
		}
		return
	}
	
	// Process actions from hub
	for _, action := range response.Actions {
		switch action.Type {
		case "EXECUTE_TASK":
			go a.executeTask(action.Task)
		case "CANCEL_TASK":
			a.executor.CancelTask(action.ExecutionID)
		case "UPDATE_CONFIG":
			a.updateConfig(action.Config)
		case "SYNC_TASKS":
			a.syncTasks()
		}
	}
}

// executeTask executes a task from hub
func (a *Agent) executeTask(taskData json.RawMessage) {
	var task executor.TaskInfo
	if err := json.Unmarshal(taskData, &task); err != nil {
		log.Printf("Failed to unmarshal task: %v", err)
		return
	}
	
	log.Printf("Executing task %s from hub", task.ExecutionID)
	
	// Report start
	a.hubClient.UpdateExecutionStatus(task.ExecutionID, "running", nil)
	
	// Execute task
	err := a.executor.ExecuteTask(a.ctx, &task)
	
	// Report completion
	if err != nil {
		a.hubClient.UpdateExecutionStatus(task.ExecutionID, "failed", err)
	} else {
		a.hubClient.UpdateExecutionStatus(task.ExecutionID, "completed", nil)
	}
	
	// Send logs
	if len(task.Logs) > 0 {
		a.hubClient.SendLogs(task.ExecutionID, task.Logs)
	}
}

// handleLocalFallback handles tasks when hub is unreachable
func (a *Agent) handleLocalFallback() {
	if a.scheduler == nil {
		return
	}
	
	log.Println("Hub unreachable, checking local tasks")
	
	// Get due tasks from local cache
	tasks := a.scheduler.GetDueTasks()
	for _, task := range tasks {
		log.Printf("Executing local fallback task: %s", task.ID)
		
		// Convert to executor task
		execTask := &executor.TaskInfo{
			ID:           task.ID,
			ExecutionID:  uuid.New().String(),
			TaskID:       task.ID,
			RemoteConfig: task.RemoteConfig,
			SourcePath:   task.SourcePath,
			DestPath:     task.DestPath,
			RcloneArgs:   task.RcloneArgs,
		}
		
		// Execute task
		if err := a.executor.ExecuteTask(a.ctx, execTask); err != nil {
			log.Printf("Local task %s failed: %v", task.ID, err)
		} else {
			log.Printf("Local task %s completed", task.ID)
		}
		
		// Update last run time
		a.scheduler.UpdateLastRun(task.ID)
	}
}

// syncTasks syncs task configurations from hub
func (a *Agent) syncTasks() {
	tasks, err := a.hubClient.GetTasks(a.ctx)
	if err != nil {
		log.Printf("Failed to sync tasks: %v", err)
		return
	}
	
	if a.scheduler != nil {
		a.scheduler.UpdateTasks(tasks)
		log.Printf("Synced %d tasks from hub", len(tasks))
	}
}

// updateConfig updates agent configuration
func (a *Agent) updateConfig(newConfig json.RawMessage) {
	var updates Config
	if err := json.Unmarshal(newConfig, &updates); err != nil {
		log.Printf("Failed to unmarshal config update: %v", err)
		return
	}
	
	// Apply safe updates (don't change critical settings like AgentID)
	if updates.MaxConcurrent > 0 {
		a.config.MaxConcurrent = updates.MaxConcurrent
	}
	if updates.HeartbeatInterval > 0 {
		a.config.HeartbeatInterval = updates.HeartbeatInterval
	}
	
	log.Println("Configuration updated from hub")
}

// ensureRegistered ensures the agent is registered with hub
func (a *Agent) ensureRegistered() error {
	// Check if already registered from config
	if a.config.AgentID != "" && a.config.APIKey != "" {
		log.Printf("Agent already registered: %s", a.config.AgentID)
		// Update the client with credentials from config
		a.hubClient.SetCredentials(a.config.AgentID, a.config.APIKey)
		return nil
	}

	// Need registration token
	if a.config.RegistrationToken == "" {
		return fmt.Errorf("not registered and no registration token provided")
	}

	log.Println("Registering with hub...")

	// Use the main hubClient (which has no credentials yet) to register
	agentID, apiKey, err := a.hubClient.Register(context.Background(), a.config.RegistrationToken, a.config.AgentName)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// Save credentials to config
	a.config.AgentID = agentID
	a.config.APIKey = apiKey

	// Update the main client with the new credentials
	a.hubClient.SetCredentials(agentID, apiKey)

	// Save to config file
	if err := a.saveConfig(); err != nil {
		log.Printf("Warning: failed to save config: %v", err)
	}

	log.Printf("Successfully registered as: %s", agentID)
	return nil
}

// saveConfig saves the current configuration to file
func (a *Agent) saveConfig() error {
	configPath := filepath.Join(a.config.WorkDir, "agent.json")
	
	data, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0600)
}

// startMetricsServer starts the metrics HTTP server
func (a *Agent) startMetricsServer() {
	// Implementation would go here
	// This would expose Prometheus metrics
	log.Printf("Metrics server started on port %d", a.config.MetricsPort)
}

// Shutdown gracefully shuts down the agent
func (a *Agent) Shutdown() {
	log.Println("Shutting down agent...")
	
	// Cancel context to stop all operations
	a.cancel()
	
	// Cancel all running tasks
	for _, task := range a.executor.GetActiveTasks() {
		a.executor.CancelTask(task.ExecutionID)
	}
	
	// Wait a bit for tasks to finish
	time.Sleep(2 * time.Second)
	
	// Final heartbeat to notify hub
	a.hubClient.SendHeartbeat(context.Background(), "offline", nil)
}

func writePIDFile(pidFile string) error {
	if pidFile == "" {
		return nil
	}
	pid := os.Getpid()
	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func removePIDFile(pidFile string) {
	if pidFile != "" {
		os.Remove(pidFile)
	}
}

// Helper functions

func printVersion() {
	fmt.Printf("Rclone Backup Agent (Standalone)\n")
	fmt.Printf("Version:    %s\n", Version)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}

func loadConfig(path string) (*Config, error) {
	// Load from environment first
	godotenv.Load()
	
	config := &Config{
		HubURL:            os.Getenv("HUB_URL"),
		RegistrationToken: os.Getenv("REGISTRATION_TOKEN"),
		AgentName:         os.Getenv("AGENT_NAME"),
		WorkDir:           os.Getenv("WORK_DIR"),
		MaxConcurrent:     3,
		HeartbeatInterval: 30,
		EnableMetrics:     true,
		MetricsPort:       9091,
	}
	
	// Set defaults
	if config.AgentName == "" {
		hostname, _ := os.Hostname()
		config.AgentName = hostname
	}
	if config.WorkDir == "" {
		config.WorkDir = "/opt/rclone-agent"
	}
	
	// Try to load from file
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}
	
	return config, nil
}

func setupLogging(config *Config) error {
	if config.LogFile != "" {
		logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		log.SetOutput(logFile)
	}
	
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	return nil
}

func handleServiceCommand(install, uninstall, start, stop bool) {
	// This would integrate with the OS service manager
	// For Linux: systemd
	// For Windows: Windows Service
	// For macOS: launchd
	
	if install {
		fmt.Println("Installing service...")
		// Implementation would go here
	}
	if uninstall {
		fmt.Println("Uninstalling service...")
		// Implementation would go here
	}
	if start {
		fmt.Println("Starting service...")
		// Implementation would go here
	}
	if stop {
		fmt.Println("Stopping service...")
		// Implementation would go here
	}
}