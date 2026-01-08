package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rclone-backup-web/hub/models"
	"github.com/rclone-backup-web/hub/services"
)

type RegisterAgentRequest struct {
	Token   string `json:"token" binding:"required"`
	Name    string `json:"name" binding:"required"`
	IsLocal bool   `json:"is_local"`
}

type RegisterAgentResponse struct {
	AgentID string `json:"agent_id"`
	APIKey  string `json:"api_key"`
}

// RegisterAgent handles new agent registration
func (h *Handler) RegisterAgent(c *gin.Context) {
	var req RegisterAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate registration token
	tokenModel := models.NewRegistrationTokenModel(h.db)
	token, err := tokenModel.GetByToken(c.Request.Context(), req.Token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid registration token"})
		return
	}

	if token.Used || time.Now().After(token.ExpiresAt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is expired or already used"})
		return
	}

	// Generate API key for the agent
	apiKey := services.GenerateAPIKey()
	apiKeyHash, err := h.authService.HashAPIKey(apiKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process API key"})
		return
	}

	// Create agent with is_local flag
	agentModel := models.NewAgentModel(h.db)
	agent, err := agentModel.Create(c.Request.Context(), req.Name, apiKeyHash, req.IsLocal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent"})
		return
	}

	// Mark token as used
	if err := tokenModel.MarkUsed(c.Request.Context(), token.ID, agent.ID); err != nil {
		// Log error but don't fail the registration
		// TODO: Add proper logging
	}

	// Send SSE event
	h.sseService.SendEvent("agent.registered", map[string]interface{}{
		"agent_id": agent.ID,
		"name":     agent.Name,
	})

	c.JSON(http.StatusCreated, RegisterAgentResponse{
		AgentID: agent.ID.String(),
		APIKey:  apiKey,
	})
}

type HeartbeatRequest struct {
	Status     string            `json:"status" binding:"required"`
	SystemInfo SystemInfoRequest `json:"system_info"`
}

type HeartbeatAction struct {
	Action      string          `json:"action"`
	Type        string          `json:"type,omitempty"`
	ExecutionID string          `json:"execution_id,omitempty"`
	TriggerMode string          `json:"trigger_mode,omitempty"`
	Task        json.RawMessage `json:"task,omitempty"`
}

type HeartbeatResponse struct {
	Actions []HeartbeatAction `json:"actions"`
}

type SystemInfoRequest struct {
	Hostname     string `json:"hostname"`
	Platform     string `json:"platform"`
	AgentVersion string `json:"agent_version"`

	CPUUsage float64 `json:"cpu_usage"`

	MemoryTotal uint64  `json:"memory_total"`
	MemoryUsed  uint64  `json:"memory_used"`
	MemoryUsage float64 `json:"memory_usage"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`

	DiskTotal uint64  `json:"disk_total"`
	DiskUsed  uint64  `json:"disk_used"`
	DiskUsage float64 `json:"disk_usage"`

	NetworkRxBytes uint64 `json:"network_rx_bytes"`
	NetworkTxBytes uint64 `json:"network_tx_bytes"`
	NetworkRxRate  uint64 `json:"network_rx_rate"`
	NetworkTxRate  uint64 `json:"network_tx_rate"`

	TCPConnections int `json:"tcp_connections"`
	UDPConnections int `json:"udp_connections"`

	ProcessCount int `json:"process_count"`
}

func (h *Handler) processAgentHeartbeat(ctx context.Context, agentID uuid.UUID, req HeartbeatRequest) (HeartbeatResponse, error) {
	agentModel := models.NewAgentModel(h.db)
	if err := agentModel.UpdateHeartbeat(ctx, agentID, req.Status); err != nil {
		return HeartbeatResponse{}, &APIError{Status: http.StatusInternalServerError, Message: "Failed to update heartbeat"}
	}

	metricsModel := models.NewMetricsModel(h.db)
	metric := &models.AgentMetric{
		AgentID:        agentID,
		Hostname:       req.SystemInfo.Hostname,
		Platform:       req.SystemInfo.Platform,
		AgentVersion:   req.SystemInfo.AgentVersion,
		CPUUsage:       req.SystemInfo.CPUUsage,
		MemoryTotal:    int64(req.SystemInfo.MemoryTotal),
		MemoryUsed:     int64(req.SystemInfo.MemoryUsed),
		MemoryUsage:    req.SystemInfo.MemoryUsage,
		SwapTotal:      int64(req.SystemInfo.SwapTotal),
		SwapUsed:       int64(req.SystemInfo.SwapUsed),
		DiskTotal:      int64(req.SystemInfo.DiskTotal),
		DiskUsed:       int64(req.SystemInfo.DiskUsed),
		DiskUsage:      req.SystemInfo.DiskUsage,
		NetworkRxBytes: int64(req.SystemInfo.NetworkRxBytes),
		NetworkTxBytes: int64(req.SystemInfo.NetworkTxBytes),
		NetworkRxRate:  int64(req.SystemInfo.NetworkRxRate),
		NetworkTxRate:  int64(req.SystemInfo.NetworkTxRate),
		TCPConnections: req.SystemInfo.TCPConnections,
		UDPConnections: req.SystemInfo.UDPConnections,
		ProcessCount:   req.SystemInfo.ProcessCount,
	}

	// Ensure values stay within signed range
	if metric.NetworkRxBytes > math.MaxInt64 {
		metric.NetworkRxBytes = math.MaxInt64
	}
	if metric.NetworkTxBytes > math.MaxInt64 {
		metric.NetworkTxBytes = math.MaxInt64
	}

	if err := metricsModel.Create(ctx, metric); err != nil {
		log.Printf("Failed to persist metrics for agent %s: %v", agentID, err)
	}

	actions := []HeartbeatAction{}

	// Check for cancelled tasks that were running on this agent
	executionModel := models.NewExecutionModel(h.db)
	cancelledExecutions, err := executionModel.GetCancelledForAgent(ctx, agentID)
	if err == nil {
		for _, exec := range cancelledExecutions {
			log.Printf("Signaling cancellation for execution %s to agent %s", exec.ID, agentID)
			actions = append(actions, HeartbeatAction{
				Action:      "CANCEL_TASK",
				Type:        "CANCEL_TASK",
				ExecutionID: exec.ID.String(),
			})
		}
	}

	// Dispatch pending FS list request (used by admin UI "browse source path" for remote agents).
	if h.fsBroker != nil {
		if req := h.fsBroker.PopNext(agentID); req != nil {
			payload, _ := json.Marshal(map[string]interface{}{
				"request_id": req.ID,
				"path":       req.Path,
				"limit":      req.Limit,
			})
			actions = append(actions, HeartbeatAction{
				Action: "FS_LIST",
				Type:   "FS_LIST",
				Task:   payload,
			})
		}
	}

	// If agent is already running a task, don't dispatch new ones
	if req.Status != "running_task" {
		taskService := services.NewTaskService(h.db)
		pendingTask, err := taskService.FindPendingTaskForAgent(ctx, agentID)

		if err != nil {
			log.Printf("Error finding pending task for agent %s: %v", agentID, err)
		} else if pendingTask != nil {
			log.Printf("Found pending task %s (%s) for agent %s", pendingTask.ID, pendingTask.Name, agentID)

			// Determine trigger mode
			triggerMode := models.TriggerModeScheduled

			// Check if there's already a pending execution (manual trigger)
			pendingExecutions, _ := executionModel.GetPendingForAgent(ctx, agentID)

			var execution *models.TaskExecution
			if len(pendingExecutions) > 0 && pendingExecutions[0].TaskID == pendingTask.ID {
				execution = pendingExecutions[0]
				triggerMode = execution.TriggerMode
			} else {
				var createErr error
				execution, createErr = taskService.CreateExecution(ctx, pendingTask.ID, agentID, triggerMode)
				if createErr != nil {
					log.Printf("Failed to create execution: %v", createErr)
					return HeartbeatResponse{Actions: actions}, nil
				}
			}

			// Build task details for agent
			taskDetails, buildErr := taskService.BuildTaskDetailsForAgent(ctx, pendingTask, execution.ID, h.cryptoService)
			if buildErr != nil {
				log.Printf("Failed to build task details: %v", buildErr)
				_ = executionModel.UpdateStatus(ctx, execution.ID, "failed", nil)
				return HeartbeatResponse{Actions: actions}, nil
			}

			taskJSON, _ := json.Marshal(taskDetails)
			actions = append(actions, HeartbeatAction{
				Action:      "EXECUTE_TASK",
				Type:        "EXECUTE_TASK",
				ExecutionID: execution.ID.String(),
				TriggerMode: triggerMode,
				Task:        taskJSON,
			})

			// Mark execution as running
			now := time.Now()
			_ = executionModel.UpdateStatus(ctx, execution.ID, "running", &now)

			log.Printf("✅ Dispatching task %s to agent %s (execution: %s, trigger: %s)",
				pendingTask.Name, agentID.String(), execution.ID.String(), triggerMode)

			h.sseService.SendEvent("task.dispatched", map[string]interface{}{
				"task_id":      pendingTask.ID.String(),
				"task_name":    pendingTask.Name,
				"agent_id":     agentID.String(),
				"execution_id": execution.ID.String(),
				"trigger_mode": triggerMode,
			})
		}
	}

	// Check if agent needs to sync config
	if h.schedulerService.NeedsConfigSync(agentID) {
		actions = append(actions, HeartbeatAction{
			Action: "SYNC_CONFIG",
			Type:   "SYNC_CONFIG",
		})
	}

	// Send SSE event for heartbeat with metrics
	h.sseService.SendEvent("agent.heartbeat", map[string]interface{}{
		"agent_id":  agentID,
		"status":    req.Status,
		"timestamp": time.Now().Format(time.RFC3339),
		"actions":   len(actions),
		"metrics": map[string]interface{}{
			"cpu_usage":       metric.CPUUsage,
			"memory_usage":    metric.MemoryUsage,
			"memory_total":    metric.MemoryTotal,
			"memory_used":     metric.MemoryUsed,
			"disk_usage":      metric.DiskUsage,
			"disk_total":      metric.DiskTotal,
			"disk_used":       metric.DiskUsed,
			"network_rx_rate": metric.NetworkRxRate,
			"network_tx_rate": metric.NetworkTxRate,
			"tcp_connections": metric.TCPConnections,
			"udp_connections": metric.UDPConnections,
			"process_count":   metric.ProcessCount,
			"recorded_at":     time.Now(),
		},
	})

	return HeartbeatResponse{Actions: actions}, nil
}

type AgentFSListResultRequest struct {
	RequestID string                 `json:"request_id" binding:"required"`
	Path      string                 `json:"path"`
	Parent    string                 `json:"parent,omitempty"`
	Entries   []services.FSListEntry `json:"entries"`
	Error     string                 `json:"error,omitempty"`
}

func (h *Handler) processAgentFSListResult(agentID uuid.UUID, req AgentFSListResultRequest) error {
	if h.fsBroker == nil {
		return &APIError{Status: http.StatusServiceUnavailable, Message: "FS broker not available"}
	}

	resp := &services.FSListResponse{
		Path:    req.Path,
		Parent:  req.Parent,
		Entries: req.Entries,
	}

	if ok := h.fsBroker.Resolve(agentID, req.RequestID, resp, strings.TrimSpace(req.Error)); !ok {
		return &APIError{Status: http.StatusNotFound, Message: "Request not found"}
	}

	return nil
}

func (h *Handler) buildLegacyAgentTasks(ctx context.Context, agentID uuid.UUID) ([]AgentLegacyTask, error) {
	taskModel := models.NewTaskModel(h.db)
	tasks, err := taskModel.GetAgentTasks(ctx, agentID)
	if err != nil {
		return nil, err
	}

	remoteModel := models.NewRemoteModel(h.db)
	remoteConfigs := make(map[uuid.UUID]string)

	legacy := make([]AgentLegacyTask, 0, len(tasks))
	for _, task := range tasks {
		if _, exists := remoteConfigs[task.RcloneRemoteID]; !exists {
			remote, err := remoteModel.GetByID(ctx, task.RcloneRemoteID)
			if err == nil {
				if decrypted, err := h.cryptoService.Decrypt(remote.ConfigData); err == nil {
					remoteConfigs[remote.ID] = services.NormalizeRcloneConfigForSingleRemote(decrypted)
				}
			}
		}

		var args []string
		if task.RcloneArgs != nil {
			_ = json.Unmarshal(task.RcloneArgs, &args)
		}

		item := AgentLegacyTask{
			ID:                task.ID.String(),
			Name:              task.Name,
			Schedule:          task.Schedule,
			RemoteConfig:      remoteConfigs[task.RcloneRemoteID],
			SourcePath:        task.SourcePath,
			DestPath:          task.DestinationPath,
			RcloneArgs:        args,
			Enabled:           task.IsActive,
			BackupMode:        task.BackupMode,
			ArchiveFormat:     task.ArchiveFormat,
			EncryptionEnabled: task.EncryptionEnabled,
		}

		if task.EncryptionEnabled {
			if task.EncryptionPasswordEnc == nil || task.EncryptionPassword2Enc == nil {
				return nil, fmt.Errorf("task %s encryption enabled but passwords missing", task.ID)
			}
			password, err := h.cryptoService.Decrypt(*task.EncryptionPasswordEnc)
			if err != nil {
				return nil, err
			}
			password2, err := h.cryptoService.Decrypt(*task.EncryptionPassword2Enc)
			if err != nil {
				return nil, err
			}
			item.EncryptionPassword = password
			item.EncryptionPassword2 = password2
		}

		legacy = append(legacy, item)
	}

	return legacy, nil
}

type UpdateExecutionRequest struct {
	Status       string `json:"status" binding:"required"`
	LogOutput    string `json:"log_output"`
	ErrorMessage string `json:"error_message,omitempty"`
	EndedAt      string `json:"ended_at"`
}

type StreamLogsRequest struct {
	Logs []struct {
		Timestamp string `json:"timestamp"`
		Message   string `json:"message"`
	} `json:"logs"`
}

func (h *Handler) processExecutionUpdate(ctx context.Context, agentID uuid.UUID, executionID uuid.UUID, req UpdateExecutionRequest) error {
	executionModel := models.NewExecutionModel(h.db)
	execution, err := executionModel.GetByID(ctx, executionID)
	if err != nil {
		return &APIError{Status: http.StatusNotFound, Message: "Execution not found"}
	}
	if execution.AgentID != agentID {
		return &APIError{Status: http.StatusForbidden, Message: "Execution does not belong to agent"}
	}

	var endedAt *time.Time
	if req.EndedAt != "" {
		t, parseErr := time.Parse(time.RFC3339, req.EndedAt)
		if parseErr == nil {
			endedAt = &t
		}
	}

	if err := executionModel.UpdateStatus(ctx, executionID, req.Status, endedAt); err != nil {
		return &APIError{Status: http.StatusInternalServerError, Message: "Failed to update execution"}
	}

	if msg := strings.TrimSpace(req.ErrorMessage); msg != "" {
		_ = executionModel.UpdateErrorMessage(ctx, executionID, msg)
	}
	if req.LogOutput != "" {
		// Never overwrite existing streamed logs.
		appendText := req.LogOutput
		if !strings.HasSuffix(appendText, "\n") {
			appendText += "\n"
		}
		_ = executionModel.AppendLogs(ctx, executionID, appendText)
	}

	h.sseService.SendEvent("execution.status.update", map[string]interface{}{
		"execution_id":  executionID,
		"status":        req.Status,
		"agent_id":      agentID,
		"error_message": strings.TrimSpace(req.ErrorMessage),
	})

	return nil
}

func (h *Handler) processExecutionLogs(ctx context.Context, agentID uuid.UUID, executionID uuid.UUID, req StreamLogsRequest) error {
	executionModel := models.NewExecutionModel(h.db)
	execution, err := executionModel.GetByID(ctx, executionID)
	if err != nil {
		return &APIError{Status: http.StatusNotFound, Message: "Execution not found"}
	}
	if execution.AgentID != agentID {
		return &APIError{Status: http.StatusForbidden, Message: "Execution does not belong to agent"}
	}

	logEntries := make([]models.LogEntry, len(req.Logs))
	for i, logEntry := range req.Logs {
		logEntries[i] = models.LogEntry{
			Timestamp: logEntry.Timestamp,
			Message:   logEntry.Message,
		}
	}

	if err := executionModel.StreamLogs(ctx, executionID, logEntries); err != nil {
		log.Printf("Failed to store execution logs: %v", err)
	}

	for _, logEntry := range req.Logs {
		h.sseService.SendEvent("execution.log.update", map[string]interface{}{
			"execution_id": executionID,
			"agent_id":     agentID,
			"log": map[string]interface{}{
				"timestamp": logEntry.Timestamp,
				"message":   logEntry.Message,
			},
		})
	}

	return nil
}
