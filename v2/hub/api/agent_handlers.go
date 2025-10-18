package api

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rclone-backup-web/hub/models"
	"github.com/rclone-backup-web/hub/services"
)

type RegisterAgentRequest struct {
	Token string `json:"token" binding:"required"`
	Name  string `json:"name" binding:"required"`
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

	// Create agent
	agentModel := models.NewAgentModel(h.db)
	agent, err := agentModel.Create(c.Request.Context(), req.Name, apiKeyHash)
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
	Status string `json:"status" binding:"required"`
}

type HeartbeatAction struct {
	Action      string          `json:"action"`
	ExecutionID string          `json:"execution_id,omitempty"`
	Task        json.RawMessage `json:"task,omitempty"`
}

type HeartbeatResponse struct {
	Actions []HeartbeatAction `json:"actions"`
}

// AgentHeartbeat handles agent heartbeat and returns pending actions
func (h *Handler) AgentHeartbeat(c *gin.Context) {
	agentID := c.MustGet("agent_id").(uuid.UUID)
	
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update agent heartbeat
	agentModel := models.NewAgentModel(h.db)
	if err := agentModel.UpdateHeartbeat(c.Request.Context(), agentID, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update heartbeat"})
		return
	}

	// Initialize actions array
	actions := []HeartbeatAction{}

	// If agent is already running a task, don't dispatch new ones
	if req.Status == "running_task" {
		log.Printf("Agent %s is busy, skipping task dispatch", agentID)
		c.JSON(http.StatusOK, HeartbeatResponse{Actions: actions})
		return
	}

	// Use TaskService to find pending tasks
	taskService := services.NewTaskService(h.db)
	pendingTask, err := taskService.FindPendingTaskForAgent(c.Request.Context(), agentID)
	
	if err != nil {
		log.Printf("Error finding pending task for agent %s: %v", agentID, err)
	} else if pendingTask != nil {
		// We have a task to execute!
		log.Printf("Found pending task %s (%s) for agent %s", 
			pendingTask.ID, pendingTask.Name, agentID)

		// Determine trigger mode
		triggerMode := "central" // Default to scheduled/central trigger
		
		// Check if there's already a pending execution (manual trigger)
		executionModel := models.NewExecutionModel(h.db)
		pendingExecutions, _ := executionModel.GetPendingForAgent(c.Request.Context(), agentID)
		
		var execution *models.TaskExecution
		if len(pendingExecutions) > 0 && pendingExecutions[0].TaskID == pendingTask.ID {
			// Use existing pending execution
			execution = pendingExecutions[0]
			triggerMode = execution.TriggerMode
		} else {
			// Create new execution for scheduled task
			execution, err = taskService.CreateExecution(c.Request.Context(), 
				pendingTask.ID, agentID, triggerMode)
			if err != nil {
				log.Printf("Failed to create execution: %v", err)
				c.JSON(http.StatusOK, HeartbeatResponse{Actions: actions})
				return
			}
		}

		// Build task details for agent
		taskDetails, err := taskService.BuildTaskDetailsForAgent(c.Request.Context(), 
			pendingTask, execution.ID, h.cryptoService)
		if err != nil {
			log.Printf("Failed to build task details: %v", err)
			// Mark execution as failed
			executionModel.UpdateStatus(c.Request.Context(), execution.ID, "failed", nil)
			c.JSON(http.StatusOK, HeartbeatResponse{Actions: actions})
			return
		}

		// Convert to JSON for HeartbeatAction
		taskJSON, _ := json.Marshal(taskDetails)

		// Create EXECUTE_TASK action
		actions = append(actions, HeartbeatAction{
			Action:      "EXECUTE_TASK",
			ExecutionID: execution.ID.String(),
			Task:        taskJSON,
		})

		// Mark execution as running
		now := time.Now()
		executionModel.UpdateStatus(c.Request.Context(), execution.ID, "running", &now)
		
		log.Printf("✅ Dispatching task %s to agent %s (execution: %s, trigger: %s)", 
			pendingTask.Name, agentID.String(), execution.ID.String(), triggerMode)
		
		// Send SSE event for real-time UI update
		h.sseService.SendEvent("task.dispatched", map[string]interface{}{
			"task_id":      pendingTask.ID.String(),
			"task_name":    pendingTask.Name,
			"agent_id":     agentID.String(),
			"execution_id": execution.ID.String(),
			"trigger_mode": triggerMode,
		})
	}

	// Check if agent needs to sync config
	if h.schedulerService.NeedsConfigSync(agentID) {
		actions = append(actions, HeartbeatAction{
			Action: "SYNC_CONFIG",
		})
	}

	// Send SSE event for heartbeat
	h.sseService.SendEvent("agent.heartbeat", map[string]interface{}{
		"agent_id": agentID,
		"status":   req.Status,
		"actions":  len(actions),
	})

	c.JSON(http.StatusOK, HeartbeatResponse{
		Actions: actions,
	})
}

// GetAgentTasks returns all tasks assigned to the agent
func (h *Handler) GetAgentTasks(c *gin.Context) {
	agentID := c.MustGet("agent_id").(uuid.UUID)

	taskModel := models.NewTaskModel(h.db)
	tasks, err := taskModel.GetAgentTasks(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tasks"})
		return
	}

	// Get remotes for each task
	remoteModel := models.NewRemoteModel(h.db)
	remotes := make(map[uuid.UUID]map[string]interface{})

	for _, task := range tasks {
		if _, exists := remotes[task.RcloneRemoteID]; !exists {
			remote, err := remoteModel.GetByID(c.Request.Context(), task.RcloneRemoteID)
			if err != nil {
				continue
			}

			// Decrypt config
			decryptedConfig, err := h.cryptoService.Decrypt(remote.ConfigData)
			if err != nil {
				continue
			}

			remotes[remote.ID] = map[string]interface{}{
				"remote_id":  remote.ID,
				"config_b64": base64.StdEncoding.EncodeToString([]byte(decryptedConfig)),
			}
		}
	}

	response := map[string]interface{}{
		"tasks":   tasks,
		"remotes": remotes,
	}

	c.JSON(http.StatusOK, response)
}

type UpdateExecutionRequest struct {
	Status    string `json:"status" binding:"required"`
	LogOutput string `json:"log_output"`
	EndedAt   string `json:"ended_at"`
}

// UpdateExecution updates the status of a task execution
func (h *Handler) UpdateExecution(c *gin.Context) {
	agentID := c.MustGet("agent_id").(uuid.UUID)
	executionIDStr := c.Param("executionId")
	
	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid execution ID"})
		return
	}

	var req UpdateExecutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify execution belongs to agent
	executionModel := models.NewExecutionModel(h.db)
	execution, err := executionModel.GetByID(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}

	if execution.AgentID != agentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Execution does not belong to agent"})
		return
	}

	// Parse ended_at time if provided
	var endedAt *time.Time
	if req.EndedAt != "" {
		t, err := time.Parse(time.RFC3339, req.EndedAt)
		if err == nil {
			endedAt = &t
		}
	}

	// Update execution
	if err := executionModel.UpdateStatus(c.Request.Context(), executionID, req.Status, endedAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update execution"})
		return
	}

	if req.LogOutput != "" {
		if err := executionModel.UpdateLogs(c.Request.Context(), executionID, req.LogOutput); err != nil {
			// Log error but don't fail the request
		}
	}

	// Send SSE event
	h.sseService.SendEvent("execution.status.update", map[string]interface{}{
		"execution_id": executionID,
		"status":       req.Status,
		"agent_id":     agentID,
	})

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

type StreamLogsRequest struct {
	Logs []struct {
		Timestamp string `json:"timestamp"`
		Message   string `json:"message"`
	} `json:"logs"`
}

// StreamExecutionLogs handles real-time log streaming from agent
func (h *Handler) StreamExecutionLogs(c *gin.Context) {
	agentID := c.MustGet("agent_id").(uuid.UUID)
	executionIDStr := c.Param("executionId")
	
	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid execution ID"})
		return
	}

	var req StreamLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify execution belongs to agent
	executionModel := models.NewExecutionModel(h.db)
	execution, err := executionModel.GetByID(c.Request.Context(), executionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}

	if execution.AgentID != agentID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Execution does not belong to agent"})
		return
	}

	// Convert to LogEntry format for database storage
	logEntries := make([]models.LogEntry, len(req.Logs))
	for i, log := range req.Logs {
		logEntries[i] = models.LogEntry{
			Timestamp: log.Timestamp,
			Message:   log.Message,
		}
	}

	// Store logs in database
	if err := executionModel.StreamLogs(c.Request.Context(), executionID, logEntries); err != nil {
		log.Printf("Failed to store execution logs: %v", err)
		// Don't fail the request, logs are best-effort
	}

	// Forward logs to SSE for real-time updates
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

	c.JSON(http.StatusAccepted, gin.H{"message": "Logs received"})