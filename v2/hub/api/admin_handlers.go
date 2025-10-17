package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rclone-backup-web/hub/models"
	"github.com/rclone-backup-web/hub/services"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
	} `json:"user"`
}

// AdminLogin handles admin login
func (h *Handler) AdminLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simple hardcoded admin check (in production, use database)
	// You should implement proper user authentication
	if req.Username != "admin" || req.Password != "admin" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := h.authService.GenerateJWT("admin-user-id", "admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	response := LoginResponse{
		Token: token,
	}
	response.User.ID = "admin-user-id"
	response.User.Name = "Admin"
	response.User.Role = "admin"

	c.JSON(http.StatusOK, response)
}

// ListAgents returns all agents
func (h *Handler) ListAgents(c *gin.Context) {
	agentModel := models.NewAgentModel(h.db)
	agents, err := agentModel.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch agents"})
		return
	}

	c.JSON(http.StatusOK, agents)
}

// DeleteAgent deletes an agent
func (h *Handler) DeleteAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	agentModel := models.NewAgentModel(h.db)
	if err := agentModel.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agent"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// CreateRegistrationToken creates a new registration token
func (h *Handler) CreateRegistrationToken(c *gin.Context) {
	token := services.GenerateRegistrationToken()
	
	tokenModel := models.NewRegistrationTokenModel(h.db)
	regToken, err := tokenModel.Create(c.Request.Context(), token, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create registration token"})
		return
	}

	c.JSON(http.StatusCreated, regToken)
}

// ListTasks returns all tasks
func (h *Handler) ListTasks(c *gin.Context) {
	taskModel := models.NewTaskModel(h.db)
	tasks, err := taskModel.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// CreateTask creates a new backup task
func (h *Handler) CreateTask(c *gin.Context) {
	var task models.BackupTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskModel := models.NewTaskModel(h.db)
	if err := taskModel.Create(c.Request.Context(), &task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	// Assign to agents if provided
	if assignedAgents := c.PostFormArray("assigned_agent_ids"); len(assignedAgents) > 0 {
		for _, agentIDStr := range assignedAgents {
			agentID, err := uuid.Parse(agentIDStr)
			if err != nil {
				continue
			}
			taskModel.AssignAgent(c.Request.Context(), task.ID, agentID)
			
			// Mark agent for config sync
			h.schedulerService.MarkAgentForSync(agentID)
		}
	}

	c.JSON(http.StatusCreated, task)
}

// UpdateTask updates a backup task
func (h *Handler) UpdateTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	var task models.BackupTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task.ID = id
	taskModel := models.NewTaskModel(h.db)
	if err := taskModel.Update(c.Request.Context(), &task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	// Mark assigned agents for config sync
	agents := task.AssignedAgents
	for _, agentID := range agents {
		h.schedulerService.MarkAgentForSync(agentID)
	}

	c.JSON(http.StatusOK, task)
}

// DeleteTask deletes a backup task
func (h *Handler) DeleteTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Get task to find assigned agents
	taskModel := models.NewTaskModel(h.db)
	task, err := taskModel.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Delete task
	if err := taskModel.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
		return
	}

	// Mark assigned agents for config sync
	for _, agentID := range task.AssignedAgents {
		h.schedulerService.MarkAgentForSync(agentID)
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListRemotes returns all remotes
func (h *Handler) ListRemotes(c *gin.Context) {
	remoteModel := models.NewRemoteModel(h.db)
	remotes, err := remoteModel.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch remotes"})
		return
	}

	c.JSON(http.StatusOK, remotes)
}

// CreateRemote creates a new remote
func (h *Handler) CreateRemote(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		ConfigData string `json:"config_data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Encrypt config data
	encryptedConfig, err := h.cryptoService.Encrypt(req.ConfigData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt config"})
		return
	}

	remoteModel := models.NewRemoteModel(h.db)
	remote, err := remoteModel.Create(c.Request.Context(), req.Name, encryptedConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create remote"})
		return
	}

	c.JSON(http.StatusCreated, remote)
}

// UpdateRemote updates a remote
func (h *Handler) UpdateRemote(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid remote ID"})
		return
	}

	var req struct {
		Name       string `json:"name" binding:"required"`
		ConfigData string `json:"config_data" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Encrypt config data
	encryptedConfig, err := h.cryptoService.Encrypt(req.ConfigData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt config"})
		return
	}

	remoteModel := models.NewRemoteModel(h.db)
	if err := remoteModel.Update(c.Request.Context(), id, req.Name, encryptedConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update remote"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

// DeleteRemote deletes a remote
func (h *Handler) DeleteRemote(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid remote ID"})
		return
	}

	remoteModel := models.NewRemoteModel(h.db)
	if err := remoteModel.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete remote"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListExecutions returns execution history
func (h *Handler) ListExecutions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	
	offset := (page - 1) * limit

	executionModel := models.NewExecutionModel(h.db)
	executions, err := executionModel.List(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch executions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": executions,
		"page":  page,
		"limit": limit,
	})
}

// GetExecutionDetail returns detailed execution information
func (h *Handler) GetExecutionDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid execution ID"})
		return
	}

	executionModel := models.NewExecutionModel(h.db)
	execution, err := executionModel.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}

	c.JSON(http.StatusOK, execution)
}

// TriggerExecution manually triggers a task execution
func (h *Handler) TriggerExecution(c *gin.Context) {
	var req struct {
		TaskID  string `json:"task_id" binding:"required"`
		AgentID string `json:"agent_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskID, err := uuid.Parse(req.TaskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	// Create execution record
	executionModel := models.NewExecutionModel(h.db)
	execution, err := executionModel.Create(c.Request.Context(), taskID, agentID, "central")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to trigger execution"})
		return
	}

	c.JSON(http.StatusCreated, execution)
}

// SSEEndpoint handles Server-Sent Events connections
func (h *Handler) SSEEndpoint(c *gin.Context) {
	clientID := h.sseService.AddClient(c)
	defer h.sseService.RemoveClient(clientID)

	h.sseService.StreamToClient(c, clientID)
}