//go:build !agent_legacy && !agent_with_logger && !agent_sidecar
// +build !agent_legacy,!agent_with_logger,!agent_sidecar

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
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
	HubURL            string `json:"hub_url"`
	RegistrationToken string `json:"registration_token,omitempty"`
	AgentID           string `json:"agent_id,omitempty"`
	APIKey            string `json:"api_key,omitempty"`

	// Agent settings
	AgentName         string `json:"agent_name"`
	WorkDir           string `json:"work_dir"`
	MaxConcurrent     int    `json:"max_concurrent"`
	HeartbeatInterval int    `json:"heartbeat_interval"`
	IsLocal           bool   `json:"is_local"`

	// Features
	EnableLocalFallback bool   `json:"enable_local_fallback"`
	EnableAutoUpdate    bool   `json:"enable_auto_update"`
	EnableMetrics       bool   `json:"enable_metrics"`
	MetricsPort         int    `json:"metrics_port"`
	EnableAPI           bool   `json:"enable_api"`
	APIPort             int    `json:"api_port"`
	APIBindAddr         string `json:"api_bind_addr"`
	APIToken            string `json:"api_token"`

	// System integration
	RunAsService bool   `json:"run_as_service"`
	LogFile      string `json:"log_file"`
	PidFile      string `json:"pid_file"`
}

// Agent represents the standalone agent
type Agent struct {
	config    *Config
	executor  *executor.TaskExecutor
	hubClient *services.HubClient
	scheduler *services.Scheduler
	apiServer *services.AgentAPIServer
	ctx       context.Context
	cancel    context.CancelFunc

	wsMu   sync.RWMutex
	wsConn *services.HubWSConn

	logQueue chan queuedLogEntry

	pendingUpdatesMu sync.Mutex
	pendingUpdates   map[string]services.WSExecutionUpdate
}

type queuedLogEntry struct {
	ExecutionID string
	Entry       services.WSLogEntry
}

type hubTaskDetails struct {
	ExecutionID         string   `json:"execution_id"`
	TaskID              string   `json:"task_id"`
	TaskName            string   `json:"task_name"`
	RemoteName          string   `json:"remote_name"`
	SourceType          string   `json:"source_type"`
	SourcePath          string   `json:"source_path"`
	DestinationPath     string   `json:"destination_path"`
	RcloneConfigB64     string   `json:"rclone_config_b64"`
	RcloneArgs          []string `json:"rclone_args"`
	BackupMode          string   `json:"backup_mode"`
	ArchiveFormat       string   `json:"archive_format"`
	EncryptionEnabled   bool     `json:"encryption_enabled"`
	EncryptionPassword  string   `json:"encryption_password"`
	EncryptionPassword2 string   `json:"encryption_password2"`
	MaxRetention        int      `json:"max_retention"`
	DBEngine            *string  `json:"db_engine"`
	DBDumpMode          *string  `json:"db_dump_mode"`
	DBHost              *string  `json:"db_host"`
	DBPort              *int     `json:"db_port"`
	DBUser              *string  `json:"db_user"`
	DBName              *string  `json:"db_name"`
	DBPassword          string   `json:"db_password"`
	DBPath              *string  `json:"db_path"`
}

type fsListActionPayload struct {
	RequestID string `json:"request_id"`
	Path      string `json:"path"`
	Limit     int    `json:"limit"`
}

func main() {
	// Parse command line flags
	var (
		configFile  = flag.String("config", "agent.json", "Configuration file path")
		showVersion = flag.Bool("version", false, "Show version information")
		install     = flag.Bool("install", false, "Install as system service")
		uninstall   = flag.Bool("uninstall", false, "Uninstall system service")
		start       = flag.Bool("start", false, "Start system service")
		stop        = flag.Bool("stop", false, "Stop system service")
		workDir     = flag.String("work-dir", "", "Override work directory")
		hubURL      = flag.String("hub-url", "", "Override hub URL")
		token       = flag.String("token", "", "Registration token for first run")
		agentName   = flag.String("name", "", "Agent name (defaults to hostname)")
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
		config:         config,
		executor:       taskExecutor,
		ctx:            ctx,
		cancel:         cancel,
		hubClient:      services.NewHubClient(config.HubURL, "", "", Version),
		logQueue:       make(chan queuedLogEntry, 8192),
		pendingUpdates: make(map[string]services.WSExecutionUpdate),
	}

	// Stream rclone output lines into a queue that can be forwarded to the Hub over WebSocket.
	taskExecutor.SetLogHook(agent.enqueueLogLine)

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

	// Start API server if enabled
	if a.config.EnableAPI {
		rclonePath := a.executor.GetRclonePath()
		authToken := a.config.APIToken
		if authToken == "" {
			authToken = a.config.APIKey
		}
		a.apiServer = services.NewAgentAPIServer(a.config.APIPort, a.config.WorkDir, rclonePath, a.config.APIBindAddr, authToken)
		go a.apiServer.Start(a.ctx)
	}

	// Start local scheduler if enabled
	if a.scheduler != nil {
		go a.scheduler.Start(a.ctx)
	}

	go a.logForwarder()

	a.hubLoop()
}

func (a *Agent) setWSConn(conn *services.HubWSConn) {
	a.wsMu.Lock()
	a.wsConn = conn
	a.wsMu.Unlock()
}

func (a *Agent) getWSConn() *services.HubWSConn {
	a.wsMu.RLock()
	conn := a.wsConn
	a.wsMu.RUnlock()
	if conn != nil && conn.IsClosed() {
		return nil
	}
	return conn
}

func (a *Agent) hubLoop() {
	interval := time.Duration(a.config.HeartbeatInterval) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}

	heartbeatTicker := time.NewTicker(interval)
	defer heartbeatTicker.Stop()

	retryBackoff := 1 * time.Second
	nextWSAttempt := time.Now()

	for {
		ws := a.getWSConn()

		var incoming <-chan services.WSMessage
		if ws != nil {
			incoming = ws.Incoming()
		}

		var reconnect <-chan time.Time
		if ws == nil {
			delay := time.Until(nextWSAttempt)
			if delay < 0 {
				delay = 0
			}
			reconnect = time.After(delay)
		}

		select {
		case <-a.ctx.Done():
			if ws != nil {
				ws.Close()
			}
			return
		case <-reconnect:
			conn, err := services.DialHubWebSocket(a.config.HubURL, a.config.AgentID, a.config.APIKey)
			if err != nil {
				log.Printf("WebSocket connect failed: %v", err)
				if retryBackoff < 30*time.Second {
					retryBackoff *= 2
				}
				nextWSAttempt = time.Now().Add(retryBackoff)
				if a.config.EnableLocalFallback {
					a.handleLocalFallback()
				}
				continue
			}

			ws = conn
			a.setWSConn(ws)
			retryBackoff = 1 * time.Second
			nextWSAttempt = time.Now()

			hb := a.hubClient.BuildHeartbeat("online", nil)
			hello := services.WSAgentHello{
				AgentVersion: Version,
				Hostname:     hb.SystemInfo.Hostname,
				Platform:     hb.SystemInfo.Platform,
			}
			_ = ws.SendJSON(services.WSMessageTypeAgentHello, hello, 2*time.Second)
			a.requestConfigSync()
			a.flushPendingExecutionUpdates(ws)
			a.sendHeartbeatWS(ws)
		case msg, ok := <-incoming:
			if !ok {
				if ws != nil {
					ws.Close()
				}
				a.setWSConn(nil)
				nextWSAttempt = time.Now().Add(retryBackoff)
				continue
			}
			a.handleWSMessage(ws, msg)
		case <-heartbeatTicker.C:
			ws = a.getWSConn()
			if ws == nil || ws.IsClosed() {
				a.setWSConn(nil)
				nextWSAttempt = time.Now()
				if a.config.EnableLocalFallback {
					a.handleLocalFallback()
				}
				continue
			}

			a.flushPendingExecutionUpdates(ws)
			a.sendHeartbeatWS(ws)
		}
	}
}

func (a *Agent) sendHeartbeatWS(ws *services.HubWSConn) {
	if ws == nil || ws.IsClosed() {
		return
	}

	activeTasks := a.executor.GetActiveTasks()
	status := "online"
	if len(activeTasks) > 0 {
		status = "running_task"
	}

	payload := a.hubClient.BuildHeartbeat(status, activeTasks)
	if err := ws.SendJSON(services.WSMessageTypeAgentHeartbeat, payload, 3*time.Second); err != nil {
		log.Printf("WebSocket heartbeat send failed: %v", err)
	}
}

func (a *Agent) handleWSMessage(ws *services.HubWSConn, msg services.WSMessage) {
	switch msg.Type {
	case services.WSMessageTypeHubPing:
		if ws != nil && !ws.IsClosed() {
			_ = ws.Send(services.WSMessage{Type: services.WSMessageTypeAgentPong}, 2*time.Second)
			a.flushPendingExecutionUpdates(ws)
			a.sendHeartbeatWS(ws)
		}
	case services.WSMessageTypeHubActions:
		var resp services.HeartbeatResponse
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			log.Printf("Failed to parse hub.actions payload: %v", err)
			return
		}
		a.handleActions(resp.Actions)
	case services.WSMessageTypeConfigSyncResponse:
		a.handleConfigSyncResponse(msg.Data)
	case services.WSMessageTypeHubError:
		log.Printf("Hub error: %s", strings.TrimSpace(string(msg.Data)))
	default:
		// Ignore unknown frames for forward compatibility.
	}
}

func (a *Agent) handleActions(actions []services.Action) {
	for _, action := range actions {
		switch strings.ToUpper(strings.TrimSpace(action.Type)) {
		case "EXECUTE_TASK":
			go a.executeTask(action.ExecutionID, action.Task)
		case "CANCEL_TASK":
			if action.ExecutionID != "" {
				a.executor.CancelTask(action.ExecutionID)
			}
		case "FS_LIST":
			go a.handleFSList(action.Task)
		case "SYNC_CONFIG", "SYNC_TASKS":
			a.requestConfigSync()
		case "UPDATE_CONFIG":
			a.updateConfig(action.Config)
		}
	}
}

func (a *Agent) requestConfigSync() {
	ws := a.getWSConn()
	if ws == nil || ws.IsClosed() {
		return
	}

	if err := ws.SendJSON(services.WSMessageTypeConfigSyncRequest, services.WSConfigSyncRequest{}, 3*time.Second); err != nil {
		log.Printf("Failed to request config sync: %v", err)
	}
}

func (a *Agent) handleConfigSyncResponse(data json.RawMessage) {
	var resp services.WSConfigSyncResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		// Backward/forward compatible: allow bare []Task payload.
		var tasks []services.Task
		if err2 := json.Unmarshal(data, &tasks); err2 != nil {
			log.Printf("Failed to parse config.sync.response: %v", err)
			return
		}
		resp.Tasks = tasks
	}

	if a.scheduler == nil {
		return
	}

	a.scheduler.UpdateTasks(resp.Tasks)
	log.Printf("Synced %d tasks from hub", len(resp.Tasks))
}

func (a *Agent) enqueueLogLine(executionID string, line string) {
	entry := queuedLogEntry{
		ExecutionID: executionID,
		Entry: services.WSLogEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Message:   line,
		},
	}

	select {
	case a.logQueue <- entry:
	default:
		// Drop if the queue is full to avoid blocking task execution.
	}
}

func (a *Agent) logForwarder() {
	flushTicker := time.NewTicker(1 * time.Second)
	defer flushTicker.Stop()

	const maxBufferedLogsPerExecution = 2000

	pending := make(map[string][]services.WSLogEntry)

	flushAll := func() {
		for executionID, entries := range pending {
			if len(entries) == 0 {
				delete(pending, executionID)
				continue
			}

			if err := a.sendExecutionLogEntries(executionID, entries); err != nil {
				continue
			}
			delete(pending, executionID)
		}
	}

	for {
		select {
		case <-a.ctx.Done():
			flushAll()
			return
		case entry := <-a.logQueue:
			pending[entry.ExecutionID] = append(pending[entry.ExecutionID], entry.Entry)
			if len(pending[entry.ExecutionID]) > maxBufferedLogsPerExecution {
				pending[entry.ExecutionID] = pending[entry.ExecutionID][len(pending[entry.ExecutionID])-maxBufferedLogsPerExecution:]
			}
			if len(pending[entry.ExecutionID]) >= 200 {
				flushAll()
			}
		case <-flushTicker.C:
			flushAll()
		}
	}
}

func (a *Agent) sendExecutionUpdate(executionID, status string, errorMessage string) {
	status = strings.ToLower(strings.TrimSpace(status))
	if executionID == "" || status == "" {
		return
	}

	payload := services.WSExecutionUpdate{
		ExecutionID:  executionID,
		Status:       status,
		ErrorMessage: strings.TrimSpace(errorMessage),
	}
	if status == "success" || status == "failed" || status == "cancelled" || status == "canceled" {
		payload.EndedAt = time.Now().Format(time.RFC3339)
	}

	ws := a.getWSConn()
	if ws != nil && !ws.IsClosed() {
		if err := ws.SendJSON(services.WSMessageTypeExecutionUpdate, payload, 3*time.Second); err == nil {
			return
		}
	}

	a.queuePendingExecutionUpdate(payload)
}

func (a *Agent) sendExecutionLogEntries(executionID string, entries []services.WSLogEntry) error {
	if executionID == "" || len(entries) == 0 {
		return nil
	}

	ws := a.getWSConn()
	if ws != nil && !ws.IsClosed() {
		payload := services.WSExecutionLogs{
			ExecutionID: executionID,
			Logs:        entries,
		}
		return ws.SendJSON(services.WSMessageTypeExecutionLogs, payload, 3*time.Second)
	}

	return services.ErrWebSocketNotReady
}

func (a *Agent) handleFSList(taskData json.RawMessage) {
	var payload fsListActionPayload
	if err := json.Unmarshal(taskData, &payload); err != nil {
		return
	}

	path := strings.TrimSpace(payload.Path)
	if path == "" {
		path = string(filepath.Separator)
	}

	limit := payload.Limit
	if limit <= 0 || limit > 2000 {
		limit = 200
	}

	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		clean = filepath.Join(a.config.WorkDir, clean)
	}

	result := services.FSListResult{
		RequestID: payload.RequestID,
		Path:      clean,
		Parent:    filepath.Dir(clean),
	}
	if result.Parent == result.Path {
		result.Parent = ""
	}

	entries, err := os.ReadDir(clean)
	if err != nil {
		result.Error = err.Error()
		a.sendFSListResult(result)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		isSymlink := entry.Type()&os.ModeSymlink != 0
		result.Entries = append(result.Entries, services.FSListEntry{
			Name:      entry.Name(),
			Path:      filepath.Join(clean, entry.Name()),
			IsDir:     true,
			IsSymlink: isSymlink,
		})
	}

	sort.Slice(result.Entries, func(i, j int) bool {
		return strings.ToLower(result.Entries[i].Name) < strings.ToLower(result.Entries[j].Name)
	})
	if len(result.Entries) > limit {
		result.Entries = result.Entries[:limit]
	}

	a.sendFSListResult(result)
}

func (a *Agent) sendFSListResult(result services.FSListResult) {
	ws := a.getWSConn()
	if ws == nil || ws.IsClosed() {
		return
	}
	_ = ws.SendJSON(services.WSMessageTypeFSListResult, result, 5*time.Second)
}

// executeTask executes a task from hub
func (a *Agent) executeTask(actionExecutionID string, taskData json.RawMessage) {
	actionExecutionID = strings.TrimSpace(actionExecutionID)

	var details hubTaskDetails
	if err := json.Unmarshal(taskData, &details); err != nil {
		log.Printf("Failed to unmarshal task payload (execution=%s): %v", actionExecutionID, err)
		if actionExecutionID != "" {
			a.sendExecutionUpdate(actionExecutionID, "failed", fmt.Sprintf("invalid task payload: %v", err))
		}
		return
	}

	if strings.TrimSpace(details.ExecutionID) == "" {
		details.ExecutionID = actionExecutionID
	}
	details.ExecutionID = strings.TrimSpace(details.ExecutionID)
	if details.ExecutionID == "" {
		log.Printf("Task payload missing execution_id (action execution=%s)", actionExecutionID)
		if actionExecutionID != "" {
			a.sendExecutionUpdate(actionExecutionID, "failed", "task payload missing execution_id")
		}
		return
	}

	details.TaskID = strings.TrimSpace(details.TaskID)
	details.TaskName = strings.TrimSpace(details.TaskName)
	details.RemoteName = strings.TrimSpace(details.RemoteName)
	details.SourceType = strings.TrimSpace(details.SourceType)
	details.SourcePath = strings.TrimSpace(details.SourcePath)
	details.DestinationPath = strings.TrimSpace(details.DestinationPath)
	details.RcloneConfigB64 = strings.TrimSpace(details.RcloneConfigB64)
	if details.MaxRetention < 0 {
		details.MaxRetention = 0
	}

	sourceType := strings.ToLower(strings.TrimSpace(details.SourceType))
	if sourceType == "" {
		sourceType = "path"
	}

	if details.TaskID == "" || details.DestinationPath == "" || details.RcloneConfigB64 == "" {
		a.sendExecutionUpdate(details.ExecutionID, "failed", "task payload missing required fields")
		return
	}

	switch sourceType {
	case "path":
		if details.SourcePath == "" {
			a.sendExecutionUpdate(details.ExecutionID, "failed", "task payload missing source_path")
			return
		}
	case "database":
		engine := ""
		if details.DBEngine != nil {
			engine = strings.ToLower(strings.TrimSpace(*details.DBEngine))
		}
		dumpMode := "single"
		if details.DBDumpMode != nil {
			if v := strings.ToLower(strings.TrimSpace(*details.DBDumpMode)); v != "" {
				dumpMode = v
			}
		}
		switch engine {
		case "postgres", "mysql":
			if details.DBHost == nil || strings.TrimSpace(*details.DBHost) == "" ||
				details.DBUser == nil || strings.TrimSpace(*details.DBUser) == "" {
				a.sendExecutionUpdate(details.ExecutionID, "failed", "task payload missing database connection fields")
				return
			}
			if dumpMode == "single" && (details.DBName == nil || strings.TrimSpace(*details.DBName) == "") {
				a.sendExecutionUpdate(details.ExecutionID, "failed", "task payload missing db_name")
				return
			}
			if dumpMode != "single" && dumpMode != "all" {
				a.sendExecutionUpdate(details.ExecutionID, "failed", "task payload has invalid db_dump_mode")
				return
			}
		case "sqlite":
			if details.DBPath == nil || strings.TrimSpace(*details.DBPath) == "" {
				a.sendExecutionUpdate(details.ExecutionID, "failed", "task payload missing db_path for sqlite")
				return
			}
			if dumpMode != "single" {
				a.sendExecutionUpdate(details.ExecutionID, "failed", "task payload has invalid db_dump_mode")
				return
			}
		default:
			a.sendExecutionUpdate(details.ExecutionID, "failed", "task payload has invalid db_engine")
			return
		}
	default:
		a.sendExecutionUpdate(details.ExecutionID, "failed", "task payload has invalid source_type")
		return
	}

	if sourceType == "database" {
		details.SourcePath = ""
		details.BackupMode = "archive"
		if strings.TrimSpace(details.ArchiveFormat) == "" {
			details.ArchiveFormat = "7z"
		}
	}

	remoteName := strings.TrimSpace(details.RemoteName)
	if remoteName == "" {
		remoteName = "remote"
	}

	destPath := strings.TrimSpace(details.DestinationPath)
	if destPath != "" && !strings.HasPrefix(destPath, "crypt:") && !looksLikeRcloneRemotePrefix(destPath) {
		prefix := remoteName + ":"
		isArchive := sourceType == "database" || strings.EqualFold(strings.TrimSpace(details.BackupMode), "archive")
		if details.EncryptionEnabled && !isArchive {
			prefix = "crypt:"
		}
		destPath = prefix + destPath
	}

	task := executor.TaskInfo{
		ID:                  details.TaskID,
		ExecutionID:         details.ExecutionID,
		TaskID:              details.TaskID,
		TaskName:            details.TaskName,
		SourceType:          sourceType,
		RemoteConfig:        details.RcloneConfigB64,
		SourcePath:          details.SourcePath,
		DestPath:            destPath,
		RcloneArgs:          details.RcloneArgs,
		BackupMode:          details.BackupMode,
		ArchiveFormat:       details.ArchiveFormat,
		EncryptionEnabled:   details.EncryptionEnabled,
		EncryptionPassword:  details.EncryptionPassword,
		EncryptionPassword2: details.EncryptionPassword2,
		MaxRetention:        details.MaxRetention,
		DBEngine:            details.DBEngine,
		DBDumpMode:          details.DBDumpMode,
		DBHost:              details.DBHost,
		DBPort:              details.DBPort,
		DBUser:              details.DBUser,
		DBName:              details.DBName,
		DBPassword:          details.DBPassword,
		DBPath:              details.DBPath,
	}

	log.Printf("Executing task %s from hub", task.ExecutionID)

	a.sendExecutionUpdate(task.ExecutionID, "running", "")

	err := a.executor.ExecuteTask(a.ctx, &task)
	if err != nil {
		status := "failed"
		if errors.Is(err, context.Canceled) || errors.Is(a.ctx.Err(), context.Canceled) {
			status = "cancelled"
		}
		a.sendExecutionUpdate(task.ExecutionID, status, err.Error())
		return
	}

	a.sendExecutionUpdate(task.ExecutionID, "success", "")
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
		destPath := strings.TrimSpace(task.DestPath)
		if destPath != "" && !strings.HasPrefix(destPath, "remote:") && !strings.HasPrefix(destPath, "crypt:") {
			prefix := "remote:"
			isArchive := strings.EqualFold(strings.TrimSpace(task.BackupMode), "archive")
			if task.EncryptionEnabled && !isArchive {
				prefix = "crypt:"
			}
			destPath = prefix + destPath
		}
		execTask := &executor.TaskInfo{
			ID:                  task.ID,
			ExecutionID:         uuid.New().String(),
			TaskID:              task.ID,
			TaskName:            strings.TrimSpace(task.Name),
			RemoteConfig:        task.RemoteConfig,
			SourcePath:          task.SourcePath,
			DestPath:            destPath,
			RcloneArgs:          task.RcloneArgs,
			BackupMode:          task.BackupMode,
			ArchiveFormat:       task.ArchiveFormat,
			EncryptionEnabled:   task.EncryptionEnabled,
			EncryptionPassword:  task.EncryptionPassword,
			EncryptionPassword2: task.EncryptionPassword2,
			MaxRetention:        task.MaxRetention,
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
	agentID, apiKey, err := a.hubClient.Register(context.Background(), a.config.RegistrationToken, a.config.AgentName, a.config.IsLocal)
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

	// Final heartbeat to notify hub (best-effort over WebSocket).
	if ws := a.getWSConn(); ws != nil {
		_ = ws.SendJSON(services.WSMessageTypeAgentHeartbeat, a.hubClient.BuildHeartbeat("offline", nil), 2*time.Second)
		ws.Close()
	}
}

func (a *Agent) queuePendingExecutionUpdate(update services.WSExecutionUpdate) {
	if strings.TrimSpace(update.ExecutionID) == "" {
		return
	}

	a.pendingUpdatesMu.Lock()
	if a.pendingUpdates == nil {
		a.pendingUpdates = make(map[string]services.WSExecutionUpdate)
	}
	a.pendingUpdates[update.ExecutionID] = update
	a.pendingUpdatesMu.Unlock()
}

func (a *Agent) flushPendingExecutionUpdates(ws *services.HubWSConn) {
	if ws == nil || ws.IsClosed() {
		return
	}

	a.pendingUpdatesMu.Lock()
	if len(a.pendingUpdates) == 0 {
		a.pendingUpdatesMu.Unlock()
		return
	}

	updates := make([]services.WSExecutionUpdate, 0, len(a.pendingUpdates))
	for _, update := range a.pendingUpdates {
		updates = append(updates, update)
	}
	a.pendingUpdates = make(map[string]services.WSExecutionUpdate)
	a.pendingUpdatesMu.Unlock()

	for i, update := range updates {
		if err := ws.SendJSON(services.WSMessageTypeExecutionUpdate, update, 3*time.Second); err != nil {
			a.pendingUpdatesMu.Lock()
			a.pendingUpdates[update.ExecutionID] = update
			for _, rest := range updates[i+1:] {
				a.pendingUpdates[rest.ExecutionID] = rest
			}
			a.pendingUpdatesMu.Unlock()
			return
		}
	}
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
		EnableAPI:         true,
		APIPort:           9092,
		APIBindAddr:       os.Getenv("API_BIND_ADDR"),
		APIToken:          os.Getenv("AGENT_API_TOKEN"),
	}

	// Set defaults
	if config.AgentName == "" {
		hostname, _ := os.Hostname()
		config.AgentName = hostname
	}
	if config.WorkDir == "" {
		config.WorkDir = "/opt/rclone-agent"
	}
	if config.APIBindAddr == "" {
		config.APIBindAddr = "127.0.0.1"
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

func looksLikeRcloneRemotePrefix(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}

	// Windows drive letter (e.g. C:\foo or C:/foo).
	if len(trimmed) >= 3 {
		c0 := trimmed[0]
		if (c0 >= 'A' && c0 <= 'Z') || (c0 >= 'a' && c0 <= 'z') {
			if trimmed[1] == ':' && (trimmed[2] == '\\' || trimmed[2] == '/') {
				return false
			}
		}
	}

	colon := strings.IndexByte(trimmed, ':')
	if colon <= 0 {
		return false
	}

	prefix := trimmed[:colon]
	if strings.ContainsAny(prefix, "/\\") {
		return false
	}

	return true
}
