package api

import (
	"encoding/base64"
	"encoding/json"
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

	// Check for pending actions
	actions := []HeartbeatAction{}

	// Check if there are any pending executions for this agent
	executionModel := models.NewExecutionModel(h.db)
	pendingExecutions, err := executionModel.GetPendingForAgent(c.Request.Context(), agentID)
	if err == nil && len(pendingExecutions) > 0 {
		for _, exec := range pendingExecutions {
			// Get task details
			taskModel := models.NewTaskModel(h.db)
			task, err := taskModel.GetByID(c.Request.Context(), exec.TaskID)
			if err != nil {
				continue
			}

			// Get remote config
			remoteModel := models.NewRemoteModel(h.db)
			remote, err := remoteModel.GetByID(c.Request.Context(), task.RcloneRemoteID)
			if err != nil {
				continue
			}

			// Decrypt remote config
			decryptedConfig, err := h.cryptoService.Decrypt(remote.ConfigData)
			if err != nil {
				continue
			}

			taskData := map[string]interface{}{
				"task_id":           task.ID,
				"remote_id":         task.RcloneRemoteID,
				"source_path":       task.SourcePath,
				"destination_path":  task.DestinationPath,
				"rclone_args":       task.RcloneArgs,
				"rclone_config_b64": base64.StdEncoding.EncodeToString([]byte(decryptedConfig)),
			}

			taskJSON, _ := json.Marshal(taskData)

			actions = append(actions, HeartbeatAction{
				Action:      "EXECUTE_TASK",
				ExecutionID: exec.ID.String(),
				Task:        taskJSON,
			})

			// Mark execution as running
			executionModel.UpdateStatus(c.Request.Context(), exec.ID, "running", nil)
		}
	}

	// Check if agent needs to sync config
	// This could be triggered by config changes, new task assignments, etc.
	if h.schedulerService.NeedsConfigSync(agentID) {
		actions = append(actions, HeartbeatAction{
			Action: "SYNC_CONFIG",
		})
	}

	// Send SSE event
	h.sseService.SendEvent("agent.heartbeat", map[string]interface{}{
		"agent_id": agentID,
		"status":   req.Status,
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

	// Forward logs to SSE
	for _, log := range req.Logs {
		h.sseService.SendEvent("execution.log.update", map[string]interface{}{
			"execution_id": executionID,
			"agent_id":     agentID,
			"log": map[string]interface{}{
				"timestamp": log.Timestamp,
				"message":   log.Message,
			},
		})
	}

	c.JSON(http.StatusAccepted, gin.H{})
}