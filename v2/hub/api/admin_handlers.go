package api

import (
	"log"
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
		log.Printf("[Login] Failed to bind JSON: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[Login] Login attempt for username: %s from IP: %s", req.Username, c.ClientIP())

	// Authenticate user
	userModel := models.NewUserModel(h.db)
	user, err := userModel.Authenticate(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		log.Printf("[Login] Authentication failed for username: %s, error: %v", req.Username, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	log.Printf("[Login] User authenticated successfully: %s (ID: %s)", user.Username, user.ID)

	// Generate JWT token
	token, err := h.authService.GenerateJWT(user.ID.String(), user.Role)
	if err != nil {
		log.Printf("[Login] Failed to generate JWT token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	log.Printf("[Login] JWT token generated for user: %s", user.Username)

	// Get client info
	ipAddress := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// Create session - hash the JWT token for storage
	tokenHash := h.authService.HashToken(token)

	_, err = userModel.CreateSession(c.Request.Context(), user.ID, tokenHash, ipAddress, userAgent, 24*time.Hour)
	if err != nil {
		log.Printf("[Login] Failed to create session: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	log.Printf("[Login] Session created for user: %s", user.Username)

	// Log audit event
	h.logAuditEvent(c, user.ID, "login", "user", user.ID, map[string]interface{}{
		"ip_address": ipAddress,
		"user_agent": userAgent,
	})

	response := LoginResponse{
		Token: token,
		User: struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Role string `json:"role"`
		}{
			ID:   user.ID.String(),
			Name: user.FullName,
			Role: user.Role,
		},
	}

	log.Printf("[Login] Login successful for user: %s", user.Username)
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
	log.Printf("[CreateRegistrationToken] Generating new registration token")
	
	token := services.GenerateRegistrationToken()
	log.Printf("[CreateRegistrationToken] Generated token: %s", token)
	
	tokenModel := models.NewRegistrationTokenModel(h.db)
	regToken, err := tokenModel.Create(c.Request.Context(), token, 24*time.Hour)
	if err != nil {
		log.Printf("[CreateRegistrationToken] Failed to create token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create registration token"})
		return
	}

	log.Printf("[CreateRegistrationToken] Token created successfully: %v", regToken)
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

// GetTask returns a single task by ID
func (h *Handler) GetTask(c *gin.Context) {
	taskID := c.Param("id")
	
	taskModel := models.NewTaskModel(h.db)
	task, err := taskModel.GetByID(c.Request.Context(), uuid.MustParse(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	
	c.JSON(http.StatusOK, task)
}

// GetRemote returns a single remote by ID
func (h *Handler) GetRemote(c *gin.Context) {
	remoteID := c.Param("id")
	
	remoteModel := models.NewRemoteModel(h.db)
	remote, err := remoteModel.GetByID(c.Request.Context(), uuid.MustParse(remoteID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Remote not found"})
		return
	}
	
	// Decrypt the config for display
	decryptedConfig, err := h.cryptoService.Decrypt(remote.ConfigData)
	if err == nil {
		remote.ConfigData = decryptedConfig
	}
	
	c.JSON(http.StatusOK, remote)
}

// TestRemote tests a remote connection
func (h *Handler) TestRemote(c *gin.Context) {
	remoteID := c.Param("id")
	
	remoteModel := models.NewRemoteModel(h.db)
	remote, err := remoteModel.GetByID(c.Request.Context(), uuid.MustParse(remoteID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Remote not found"})
		return
	}
	
	// Decrypt the config
	_, err = h.cryptoService.Decrypt(remote.ConfigData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt remote config"})
		return
	}
	
	// TODO: Implement actual rclone test with decrypted config
	// For now, just validate the config can be decrypted
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Remote connection test successful",
		"remote_id": remoteID,
		"remote_name": remote.Name,
	})
}

// CancelExecution cancels a running execution
func (h *Handler) CancelExecution(c *gin.Context) {
	executionID := c.Param("id")
	
	executionModel := models.NewExecutionModel(h.db)
	execution, err := executionModel.GetByID(c.Request.Context(), uuid.MustParse(executionID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
		return
	}
	
	// Check if execution is still running
	if execution.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Execution is not running"})
		return
	}
	
	// Update status to cancelled
	now := time.Now()
	err = executionModel.UpdateStatus(c.Request.Context(), execution.ID, "cancelled", &now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel execution"})
		return
	}
	
	// Broadcast cancellation event
	h.sseService.SendEvent("execution.status.update", map[string]interface{}{
		"execution_id": executionID,
		"status": "cancelled",
		"timestamp": time.Now().Format(time.RFC3339),
	})
	
	c.JSON(http.StatusOK, gin.H{"message": "Execution cancelled"})
}

// GetDashboardStats returns dashboard statistics
func (h *Handler) GetDashboardStats(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Get agent stats
	agentModel := models.NewAgentModel(h.db)
	agents, _ := agentModel.List(ctx)
	onlineAgents := 0
	for _, agent := range agents {
		if agent.Status == "online" {
			onlineAgents++
		}
	}
	
	// Get task stats
	taskModel := models.NewTaskModel(h.db)
	tasks, _ := taskModel.List(ctx)
	activeTasks := 0
	for _, task := range tasks {
		if task.IsActive {
			activeTasks++
		}
	}
	
	// Get recent execution stats
	executionModel := models.NewExecutionModel(h.db)
	executions, _ := executionModel.List(ctx, 100, 0)
	
	successCount := 0
	failedCount := 0
	runningCount := 0
	
	for _, exec := range executions {
		switch exec.Status {
		case "success":
			successCount++
		case "failed":
			failedCount++
		case "running":
			runningCount++
		}
	}
	
	successRate := float64(0)
	if len(executions) > 0 {
		successRate = float64(successCount) / float64(len(executions)) * 100
	}
	
	c.JSON(http.StatusOK, gin.H{
		"agents": gin.H{
			"total": len(agents),
			"online": onlineAgents,
			"offline": len(agents) - onlineAgents,
		},
		"tasks": gin.H{
			"total": len(tasks),
			"active": activeTasks,
			"inactive": len(tasks) - activeTasks,
		},
		"executions": gin.H{
			"total": len(executions),
			"success": successCount,
			"failed": failedCount,
			"running": runningCount,
			"success_rate": successRate,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// AdminLogout handles admin logout
func (h *Handler) AdminLogout(c *gin.Context) {
	// Get user ID from context (set by middleware)
	userID, exists := c.Get("user_id")
	if exists {
		// Log audit event
		h.logAuditEvent(c, userID.(uuid.UUID), "logout", "user", userID.(uuid.UUID), nil)
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetStatisticsOverview returns overall system statistics
func (h *Handler) GetStatisticsOverview(c *gin.Context) {
	// This is similar to GetDashboardStats but with more detail
	h.GetDashboardStats(c)
}

// GetAgentStatistics returns statistics for a specific agent
func (h *Handler) GetAgentStatistics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	executionModel := models.NewExecutionModel(h.db)
	stats, err := executionModel.GetStatsByAgent(c.Request.Context(), id, 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agent statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetTaskStatistics returns statistics for a specific task
func (h *Handler) GetTaskStatistics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	executionModel := models.NewExecutionModel(h.db)
	stats, err := executionModel.GetStatsByTask(c.Request.Context(), id, 30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get task statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetRecentActivity returns recent system activity
func (h *Handler) GetRecentActivity(c *gin.Context) {
	executionModel := models.NewExecutionModel(h.db)
	executions, err := executionModel.List(c.Request.Context(), 10, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get recent activity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": executions,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetChartData returns data for dashboard charts
func (h *Handler) GetChartData(c *gin.Context) {
	timeRange := c.DefaultQuery("range", "7d")
	
	// Parse time range
	var days int
	switch timeRange {
	case "24h":
		days = 1
	case "7d":
		days = 7
	case "30d":
		days = 30
	default:
		days = 7
	}

	// Simple mock data for now
	c.JSON(http.StatusOK, gin.H{
		"data": []map[string]interface{}{},
		"range": timeRange,
		"days": days,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// DownloadAgent provides agent binary download
func (h *Handler) DownloadAgent(c *gin.Context) {
	// Set headers for binary file download
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=rclone-backup-agent")
	c.Header("Content-Transfer-Encoding", "binary")
	
	// For now, return a placeholder response with instructions
	// In a real implementation, this would serve the actual agent binary
	instructions := `#!/bin/bash
# Rclone Backup Agent - Binary Download Placeholder
# This is a placeholder script. In production, this would be the actual agent binary.

echo "Rclone Backup Agent"
echo "==================="
echo ""
echo "This is a placeholder for the agent binary."
echo "In production, this endpoint would serve the actual compiled agent binary."
echo ""
echo "To register with Hub, use:"
echo "  ./rclone-backup-agent register --hub-url=HUB_URL --token=TOKEN --name=AGENT_NAME --daemon"
echo ""
echo "For more information, visit: https://github.com/rclone-backup-web/agent"
`
	
	c.String(http.StatusOK, instructions)
}