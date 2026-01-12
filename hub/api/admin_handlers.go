package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	// Enforce admin role
	if !user.IsAdmin && user.Role != "admin" {
		log.Printf("[Login] User %s is not admin, role: %s", user.Username, user.Role)
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin privileges required"})
		return
	}

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

// ListAgentDirectory lists directories on an Agent filesystem.
// Local agents use LOCAL_AGENT_URL; remote agents use the Agent WebSocket connection.
func (h *Handler) ListAgentDirectory(c *gin.Context) {
	idStr := c.Param("id")
	agentID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	agentModel := models.NewAgentModel(h.db)
	agent, err := agentModel.GetByID(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	path := strings.TrimSpace(c.Query("path"))
	if path == "" {
		path = "/"
	}

	limit := 200
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 2000 {
			limit = v
		}
	}

	if agent.IsLocal {
		result, err := h.rcloneService.ListDirectory(c.Request.Context(), path, limit)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"error":   "Failed to list directory",
				"message": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, result)
		return
	}

	if agent.Status == "offline" {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Agent offline",
			"message": "Directory listing requires an online agent.",
		})
		return
	}

	if h.fsBroker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "FS broker not available"})
		return
	}

	if h.wsService == nil || !h.wsService.IsConnected(agentID) {
		c.JSON(http.StatusConflict, gin.H{
			"error":   "Agent not connected",
			"message": "Directory listing requires an active WebSocket connection to the agent.",
		})
		return
	}

	req, resultCh := h.fsBroker.Enqueue(agentID, path, limit)
	log.Printf("[FSList] enqueue agent=%s request=%s path=%q limit=%d", agentID.String(), req.ID, req.Path, req.Limit)

	payload, _ := json.Marshal(map[string]interface{}{
		"request_id": req.ID,
		"path":       req.Path,
		"limit":      req.Limit,
	})
	
	resp := HeartbeatResponse{
		Actions: []HeartbeatAction{{
			Action: "FS_LIST",
			Type:   "FS_LIST",
			Task:   payload,
		}},
	}
	respData, _ := json.Marshal(resp)
	
	wsMsg := WSMessage{
		Type: WSMessageTypeHubActions,
		Data: respData,
	}
	
	// Use longer timeout for FS_LIST requests (10s instead of default 3s)
	// and handle send failure immediately instead of waiting for response timeout
	msgData, _ := json.Marshal(wsMsg)
	if err := h.wsService.SendBytesWithTimeout(agentID, msgData, 10*time.Second); err != nil {
		log.Printf("[FSList] dispatch failed agent=%s request=%s: %v", agentID.String(), req.ID, err)
		h.fsBroker.Cancel(agentID, req.ID)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Failed to send request to agent",
			"message": err.Error(),
		})
		return
	}
	h.fsBroker.MarkDispatched(agentID, req.ID)

	waitCtx, cancel := context.WithTimeout(c.Request.Context(), 40*time.Second)
	defer cancel()

	select {
	case result, ok := <-resultCh:
		if !ok || result.Response == nil {
			log.Printf("[FSList] timeout agent=%s request=%s duration=%s", agentID.String(), req.ID, time.Since(req.CreatedAt))
			c.JSON(http.StatusGatewayTimeout, gin.H{
				"error":   "Directory listing timed out",
				"message": "Agent did not respond in time.",
			})
			return
		}
		if strings.TrimSpace(result.Error) != "" {
			log.Printf("[FSList] error agent=%s request=%s path=%q duration=%s err=%q", agentID.String(), req.ID, req.Path, time.Since(req.CreatedAt), strings.TrimSpace(result.Error))
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Failed to list directory",
				"message": result.Error,
			})
			return
		}
		log.Printf("[FSList] ok agent=%s request=%s path=%q duration=%s entries=%d", agentID.String(), req.ID, req.Path, time.Since(req.CreatedAt), len(result.Response.Entries))
		c.JSON(http.StatusOK, result.Response)
		return
	case <-waitCtx.Done():
		h.fsBroker.Cancel(agentID, req.ID)
		log.Printf("[FSList] timeout agent=%s request=%s duration=%s", agentID.String(), req.ID, time.Since(req.CreatedAt))
		c.JSON(http.StatusGatewayTimeout, gin.H{
			"error":   "Directory listing timed out",
			"message": "Agent did not respond in time.",
		})
		return
	}
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

	// Check if agent is local (cannot be deleted)
	agent, err := agentModel.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	if agent.IsLocal {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete local agent"})
		return
	}

	if err := agentModel.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agent"})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// UpdateAgent updates an agent's information
func (h *Handler) UpdateAgent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agentModel := models.NewAgentModel(h.db)
	if err := agentModel.UpdateName(c.Request.Context(), id, req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agent"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Agent updated successfully"})
}

func (h *Handler) GetAgentMetricsLatest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	metricsModel := models.NewMetricsModel(h.db)
	metric, err := metricsModel.GetLatest(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "No metrics found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch metrics"})
		return
	}

	c.JSON(http.StatusOK, metric)
}

func (h *Handler) GetAgentMetricsHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid agent ID"})
		return
	}

	hoursParam := c.DefaultQuery("hours", "24")
	hours, err := strconv.ParseFloat(hoursParam, 64)
	if err != nil || hours <= 0 {
		hours = 24
	}

	end := time.Now()
	start := end.Add(-time.Duration(hours * float64(time.Hour)))

	metricsModel := models.NewMetricsModel(h.db)
	history, err := metricsModel.GetHistory(c.Request.Context(), id, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch metrics history"})
		return
	}

	c.JSON(http.StatusOK, history)
}

// CreateRegistrationToken creates a new registration token
func (h *Handler) CreateRegistrationToken(c *gin.Context) {
	log.Printf("[CreateRegistrationToken] Generating new registration token")

	token := services.GenerateRegistrationToken()
	if h.logTokens {
		log.Printf("[CreateRegistrationToken] Generated token: %s", token)
	} else {
		log.Printf("[CreateRegistrationToken] Generated token (redacted)")
	}

	tokenModel := models.NewRegistrationTokenModel(h.db)
	regToken, err := tokenModel.Create(c.Request.Context(), token, 24*time.Hour)
	if err != nil {
		log.Printf("[CreateRegistrationToken] Failed to create token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create registration token"})
		return
	}

	if h.logTokens {
		log.Printf("[CreateRegistrationToken] Token created successfully: %v", regToken)
	} else {
		log.Printf("[CreateRegistrationToken] Token created successfully: id=%s expires_at=%s", regToken.ID, regToken.ExpiresAt.Format(time.RFC3339))
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

type TaskUpsertRequest struct {
	Name            string   `json:"name" binding:"required"`
	RcloneRemoteID  string   `json:"rclone_remote_id" binding:"required"`
	SourceType      string   `json:"source_type,omitempty"`
	SourcePath      string   `json:"source_path,omitempty"`
	DBEngine        string   `json:"db_engine,omitempty"`
	DBDumpMode      string   `json:"db_dump_mode,omitempty"`
	DBHost          string   `json:"db_host,omitempty"`
	DBPort          *int     `json:"db_port,omitempty"`
	DBUser          string   `json:"db_user,omitempty"`
	DBName          string   `json:"db_name,omitempty"`
	DBPassword      string   `json:"db_password,omitempty"`
	DBPath          string   `json:"db_path,omitempty"`
	DestinationPath string   `json:"destination_path" binding:"required"`
	Schedule        string   `json:"schedule" binding:"required"`
	RcloneArgs      []string `json:"rclone_args"`
	IsActive        bool     `json:"is_active"`

	BackupMode    string `json:"backup_mode,omitempty"`
	ArchiveFormat string `json:"archive_format,omitempty"`

	EncryptionEnabled  bool   `json:"encryption_enabled"`
	EncryptionPassword string `json:"encryption_password,omitempty"`

	RetentionDays *int `json:"retention_days,omitempty"`
	MaxRetention  *int `json:"max_retention,omitempty"`

	AssignedAgentIDs []string `json:"assigned_agent_ids,omitempty"`
	AssignedAgents   []string `json:"assigned_agents,omitempty"`
}

func normalizeBackupMode(mode string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "sync":
		return "sync", true
	case "archive":
		return "archive", true
	default:
		return "", false
	}
}

func normalizeArchiveFormat(format string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "tar.gz", "tgz":
		return "tar.gz", true
	case "zip":
		return "zip", true
	case "7z":
		return "7z", true
	default:
		return "", false
	}
}

func normalizeSourceType(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "path", "file", "filesystem":
		return "path", true
	case "database", "db":
		return "database", true
	default:
		return "", false
	}
}

func normalizeDBEngine(engine string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(engine)) {
	case "postgres", "postgresql", "pgsql":
		return "postgres", true
	case "mysql", "mariadb":
		return "mysql", true
	case "sqlite", "sqlite3":
		return "sqlite", true
	default:
		return "", false
	}
}

func normalizeDBDumpMode(mode string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "single", "one":
		return "single", true
	case "all", "all_databases", "alldatabases":
		return "all", true
	default:
		return "", false
	}
}

// looksLikeRcloneRemotePrefix returns true if the path starts with "<name>:" where
// "<name>" has no path separators. It intentionally allows Windows absolute paths
// like "C:\\foo" or "C:/foo" (drive letter).
func looksLikeRcloneRemotePrefix(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}

	colon := strings.IndexByte(path, ':')
	if colon <= 0 {
		return false
	}

	// Allow Windows drive letter absolute paths (e.g. C:\ or C:/).
	if colon == 1 && len(path) >= 3 {
		drive := path[0]
		next := path[2]
		if (drive >= 'A' && drive <= 'Z') || (drive >= 'a' && drive <= 'z') {
			if next == '\\' || next == '/' {
				return false
			}
		}
	}

	prefix := path[:colon]
	if strings.ContainsAny(prefix, "/\\") {
		return false
	}

	return true
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if _, ok := seen[raw]; ok {
			continue
		}
		seen[raw] = struct{}{}
		out = append(out, raw)
	}
	return out
}

func uniqueUUIDs(values []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(values))
	out := make([]uuid.UUID, 0, len(values))
	for _, v := range values {
		if v == uuid.Nil {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func parseUUIDList(values []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(values))
	for _, raw := range values {
		id, err := uuid.Parse(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("invalid uuid: %s", raw)
		}
		out = append(out, id)
	}
	return out, nil
}

func deriveTaskPassword2() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CreateTask creates a new backup task
func (h *Handler) CreateTask(c *gin.Context) {
	var req TaskUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if looksLikeRcloneRemotePrefix(req.DestinationPath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "destination_path must not include remote prefix (e.g. 's3:')"})
		return
	}

	remoteID, err := uuid.Parse(req.RcloneRemoteID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rclone_remote_id"})
		return
	}

	sourceType, ok := normalizeSourceType(req.SourceType)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source_type"})
		return
	}

	sourcePath := strings.TrimSpace(req.SourcePath)
	if sourceType == "path" && sourcePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_path is required when source_type is path"})
		return
	}
	if sourceType == "database" {
		sourcePath = ""
	}

	backupMode := "archive"
	if sourceType == "path" {
		var ok bool
		backupMode, ok = normalizeBackupMode(req.BackupMode)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid backup_mode"})
			return
		}
	}

	archiveFormat := "7z"
	if sourceType == "path" {
		var ok bool
		archiveFormat, ok = normalizeArchiveFormat(req.ArchiveFormat)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid archive_format"})
			return
		}
		if backupMode == "archive" && req.EncryptionEnabled {
			archiveFormat = "7z"
		}
	}

	rcloneArgs, err := json.Marshal(req.RcloneArgs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rclone_args"})
		return
	}

	var passwordEnc *string
	var password2Enc *string
	if req.EncryptionEnabled {
		if strings.TrimSpace(req.EncryptionPassword) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "encryption_password is required when encryption_enabled is true"})
			return
		}

		password2, err := deriveTaskPassword2()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate encryption_password2"})
			return
		}

		encryptedPassword, err := h.cryptoService.Encrypt(strings.TrimSpace(req.EncryptionPassword))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt encryption_password"})
			return
		}
		encryptedPassword2, err := h.cryptoService.Encrypt(password2)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt encryption_password2"})
			return
		}
		passwordEnc = &encryptedPassword
		password2Enc = &encryptedPassword2
	}

	var (
		dbEngine      *string
		dbDumpMode    *string
		dbHost        *string
		dbPort        *int
		dbUser        *string
		dbName        *string
		dbPath        *string
		dbPasswordEnc *string
	)

	if sourceType == "database" {
		engine, ok := normalizeDBEngine(req.DBEngine)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid db_engine"})
			return
		}
		dbEngine = &engine

		dumpMode, ok := normalizeDBDumpMode(req.DBDumpMode)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid db_dump_mode"})
			return
		}
		if engine == "sqlite" && dumpMode != "single" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "db_dump_mode must be single for sqlite"})
			return
		}
		dbDumpMode = &dumpMode

		if engine == "sqlite" {
			path := strings.TrimSpace(req.DBPath)
			if path == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "db_path is required when db_engine is sqlite"})
				return
			}
			dbPath = &path
		} else {
			host := strings.TrimSpace(req.DBHost)
			user := strings.TrimSpace(req.DBUser)
			name := strings.TrimSpace(req.DBName)
			if host == "" || user == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "db_host and db_user are required for database tasks"})
				return
			}
			if dumpMode == "single" && name == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "db_name is required when db_dump_mode is single"})
				return
			}

			port := 3306
			if engine == "postgres" {
				port = 5432
			}
			if req.DBPort != nil && *req.DBPort > 0 {
				port = *req.DBPort
			}

			dbHost = &host
			dbUser = &user
			if name != "" {
				dbName = &name
			}
			dbPort = &port
		}

		if raw := strings.TrimSpace(req.DBPassword); raw != "" {
			enc, err := h.cryptoService.Encrypt(raw)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt db_password"})
				return
			}
			dbPasswordEnc = &enc
		}
	}

	maxRetention := req.MaxRetention
	if maxRetention != nil && *maxRetention <= 0 {
		maxRetention = nil
	}

	task := models.BackupTask{
		Name:                   req.Name,
		RcloneRemoteID:         remoteID,
		SourceType:             sourceType,
		SourcePath:             sourcePath,
		DBEngine:               dbEngine,
		DBDumpMode:             dbDumpMode,
		DBHost:                 dbHost,
		DBPort:                 dbPort,
		DBUser:                 dbUser,
		DBName:                 dbName,
		DBPath:                 dbPath,
		DBPasswordEnc:          dbPasswordEnc,
		DestinationPath:        req.DestinationPath,
		Schedule:               req.Schedule,
		RcloneArgs:             rcloneArgs,
		IsActive:               req.IsActive,
		BackupMode:             backupMode,
		ArchiveFormat:          archiveFormat,
		EncryptionEnabled:      req.EncryptionEnabled,
		EncryptionPasswordEnc:  passwordEnc,
		EncryptionPassword2Enc: password2Enc,
		RetentionDays:          req.RetentionDays,
		MaxRetention:           maxRetention,
	}

	taskModel := models.NewTaskModel(h.db)
	if err := taskModel.Create(c.Request.Context(), &task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	assignedRaw := uniqueStrings(append(req.AssignedAgentIDs, req.AssignedAgents...))
	if len(assignedRaw) > 0 {
		agentIDs, err := parseUUIDList(assignedRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		task.AssignedAgents = agentIDs
		for _, agentID := range agentIDs {
			_ = taskModel.AssignAgent(c.Request.Context(), task.ID, agentID)
			h.schedulerService.MarkAgentForSync(agentID)
			if h.wsService != nil && h.wsService.IsConnected(agentID) {
				_ = h.wsService.SendJSON(agentID, WSMessage{Type: WSMessageTypeHubPing})
			}
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

	var req TaskUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if looksLikeRcloneRemotePrefix(req.DestinationPath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "destination_path must not include remote prefix (e.g. 's3:')"})
		return
	}

	taskModel := models.NewTaskModel(h.db)

	current, err := taskModel.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	remoteID, err := uuid.Parse(req.RcloneRemoteID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rclone_remote_id"})
		return
	}

	sourceTypeInput := strings.TrimSpace(req.SourceType)
	if sourceTypeInput == "" {
		sourceTypeInput = strings.TrimSpace(current.SourceType)
	}
	sourceType, ok := normalizeSourceType(sourceTypeInput)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source_type"})
		return
	}

	sourcePath := strings.TrimSpace(req.SourcePath)
	if sourceType == "path" {
		if sourcePath == "" {
			sourcePath = strings.TrimSpace(current.SourcePath)
		}
		if sourcePath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source_path is required when source_type is path"})
			return
		}
	} else {
		sourcePath = ""
	}

	backupMode := "archive"
	if sourceType == "path" {
		backupModeInput := strings.TrimSpace(req.BackupMode)
		if backupModeInput == "" {
			backupModeInput = strings.TrimSpace(current.BackupMode)
		}
		var ok bool
		backupMode, ok = normalizeBackupMode(backupModeInput)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid backup_mode"})
			return
		}
	}

	archiveFormat := "7z"
	if sourceType == "path" {
		archiveFormatInput := strings.TrimSpace(req.ArchiveFormat)
		if archiveFormatInput == "" {
			archiveFormatInput = strings.TrimSpace(current.ArchiveFormat)
		}
		var ok bool
		archiveFormat, ok = normalizeArchiveFormat(archiveFormatInput)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid archive_format"})
			return
		}
		if backupMode == "archive" && req.EncryptionEnabled {
			archiveFormat = "7z"
		}
	}

	rcloneArgs, err := json.Marshal(req.RcloneArgs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rclone_args"})
		return
	}

	var (
		dbEngine      *string
		dbDumpMode    *string
		dbHost        *string
		dbPort        *int
		dbUser        *string
		dbName        *string
		dbPath        *string
		dbPasswordEnc *string
	)
	if sourceType == "database" {
		engineInput := strings.TrimSpace(req.DBEngine)
		if engineInput == "" && current.DBEngine != nil {
			engineInput = strings.TrimSpace(*current.DBEngine)
		}
		engine, ok := normalizeDBEngine(engineInput)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid db_engine"})
			return
		}
		dbEngine = &engine

		dumpModeInput := strings.TrimSpace(req.DBDumpMode)
		if dumpModeInput == "" && current.DBDumpMode != nil {
			dumpModeInput = strings.TrimSpace(*current.DBDumpMode)
		}
		dumpMode, ok := normalizeDBDumpMode(dumpModeInput)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid db_dump_mode"})
			return
		}
		if engine == "sqlite" && dumpMode != "single" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "db_dump_mode must be single for sqlite"})
			return
		}
		dbDumpMode = &dumpMode

		if engine == "sqlite" {
			path := strings.TrimSpace(req.DBPath)
			if path == "" && current.DBPath != nil {
				path = strings.TrimSpace(*current.DBPath)
			}
			if path == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "db_path is required when db_engine is sqlite"})
				return
			}
			dbPath = &path
		} else {
			host := strings.TrimSpace(req.DBHost)
			user := strings.TrimSpace(req.DBUser)
			name := strings.TrimSpace(req.DBName)
			if host == "" && current.DBHost != nil {
				host = strings.TrimSpace(*current.DBHost)
			}
			if user == "" && current.DBUser != nil {
				user = strings.TrimSpace(*current.DBUser)
			}
			if name == "" && current.DBName != nil {
				name = strings.TrimSpace(*current.DBName)
			}
			if host == "" || user == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "db_host and db_user are required for database tasks"})
				return
			}
			if dumpMode == "single" && name == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "db_name is required when db_dump_mode is single"})
				return
			}

			port := 3306
			if engine == "postgres" {
				port = 5432
			}
			if current.DBPort != nil && *current.DBPort > 0 {
				port = *current.DBPort
			}
			if req.DBPort != nil && *req.DBPort > 0 {
				port = *req.DBPort
			}

			dbHost = &host
			dbUser = &user
			if name != "" {
				dbName = &name
			} else if dumpMode == "single" && current.DBName != nil {
				dbName = current.DBName
			}
			dbPort = &port
		}

		if raw := strings.TrimSpace(req.DBPassword); raw != "" {
			enc, err := h.cryptoService.Encrypt(raw)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt db_password"})
				return
			}
			dbPasswordEnc = &enc
		} else if strings.EqualFold(strings.TrimSpace(current.SourceType), "database") {
			dbPasswordEnc = current.DBPasswordEnc
		}
	}

	next := models.BackupTask{
		ID:              id,
		Name:            req.Name,
		RcloneRemoteID:  remoteID,
		SourceType:      sourceType,
		SourcePath:      sourcePath,
		DBEngine:        dbEngine,
		DBDumpMode:      dbDumpMode,
		DBHost:          dbHost,
		DBPort:          dbPort,
		DBUser:          dbUser,
		DBName:          dbName,
		DBPath:          dbPath,
		DBPasswordEnc:   dbPasswordEnc,
		DestinationPath: req.DestinationPath,
		Schedule:        req.Schedule,
		RcloneArgs:      rcloneArgs,
		IsActive:        req.IsActive,
		BackupMode:      backupMode,
		ArchiveFormat:   archiveFormat,
		RetentionDays:   current.RetentionDays,
		MaxRetention:    current.MaxRetention,
	}

	if req.RetentionDays != nil {
		next.RetentionDays = req.RetentionDays
	}
	if req.MaxRetention != nil {
		if *req.MaxRetention > 0 {
			next.MaxRetention = req.MaxRetention
		} else {
			next.MaxRetention = nil
		}
	}

	if req.EncryptionEnabled {
		next.EncryptionEnabled = true
		if strings.TrimSpace(req.EncryptionPassword) != "" {
			password2, err := deriveTaskPassword2()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate encryption_password2"})
				return
			}

			encryptedPassword, err := h.cryptoService.Encrypt(strings.TrimSpace(req.EncryptionPassword))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt encryption_password"})
				return
			}
			encryptedPassword2, err := h.cryptoService.Encrypt(password2)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt encryption_password2"})
				return
			}
			next.EncryptionPasswordEnc = &encryptedPassword
			next.EncryptionPassword2Enc = &encryptedPassword2
		} else {
			if current.EncryptionPasswordEnc == nil || current.EncryptionPassword2Enc == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "encryption_password is required when enabling encryption for the first time"})
				return
			}
			next.EncryptionPasswordEnc = current.EncryptionPasswordEnc
			next.EncryptionPassword2Enc = current.EncryptionPassword2Enc
		}
	} else {
		next.EncryptionEnabled = false
		next.EncryptionPasswordEnc = nil
		next.EncryptionPassword2Enc = nil
	}

	if err := taskModel.Update(c.Request.Context(), &next); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
		return
	}

	// Update agent assignments if the request provided them (nil slice means not provided).
	var desiredRaw []string
	if req.AssignedAgentIDs != nil {
		desiredRaw = req.AssignedAgentIDs
	} else if req.AssignedAgents != nil {
		desiredRaw = req.AssignedAgents
	}

	oldAgentIDs := current.AssignedAgents
	newAgentIDs := oldAgentIDs

	if desiredRaw != nil {
		desiredRaw = uniqueStrings(desiredRaw)
		parsed, err := parseUUIDList(desiredRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		newAgentIDs = parsed

		desiredSet := make(map[uuid.UUID]struct{}, len(newAgentIDs))
		for _, id := range newAgentIDs {
			desiredSet[id] = struct{}{}
		}
		currentSet := make(map[uuid.UUID]struct{}, len(oldAgentIDs))
		for _, id := range oldAgentIDs {
			currentSet[id] = struct{}{}
		}

		for _, agentID := range oldAgentIDs {
			if _, ok := desiredSet[agentID]; ok {
				continue
			}
			_ = taskModel.UnassignAgent(c.Request.Context(), id, agentID)
		}
		for _, agentID := range newAgentIDs {
			if _, ok := currentSet[agentID]; ok {
				continue
			}
			_ = taskModel.AssignAgent(c.Request.Context(), id, agentID)
		}
	}

	// Mark affected agents for config sync (both removed and added agents).
	for _, agentID := range uniqueUUIDs(append(oldAgentIDs, newAgentIDs...)) {
		h.schedulerService.MarkAgentForSync(agentID)
		if h.wsService != nil && h.wsService.IsConnected(agentID) {
			_ = h.wsService.SendJSON(agentID, WSMessage{Type: WSMessageTypeHubPing})
		}
	}

	updated, err := taskModel.GetByID(c.Request.Context(), id)
	if err != nil {
		next.AssignedAgents = newAgentIDs
		c.JSON(http.StatusOK, next)
		return
	}
	c.JSON(http.StatusOK, updated)
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
		Name       string  `json:"name" binding:"required"`
		ConfigData string  `json:"config_data" binding:"required"`
		Type       *string `json:"type,omitempty"`
		PresetKey  *string `json:"preset_key,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	remoteName, err := validateRemoteName(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	remoteType, err := validateRemoteConfig(remoteName, req.ConfigData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Type != nil && strings.TrimSpace(*req.Type) != "" && strings.TrimSpace(*req.Type) != remoteType {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type does not match config_data"})
		return
	}

	if req.PresetKey != nil && strings.TrimSpace(*req.PresetKey) != "" {
		if err := validateRemotePresetConfig(strings.TrimSpace(*req.PresetKey), remoteName, req.ConfigData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Encrypt config data
	encryptedConfig, err := h.cryptoService.Encrypt(req.ConfigData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt config"})
		return
	}

	remoteModel := models.NewRemoteModel(h.db)
	remote, err := remoteModel.Create(c.Request.Context(), remoteName, encryptedConfig, &remoteType)
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
		Name       string  `json:"name" binding:"required"`
		ConfigData string  `json:"config_data" binding:"required"`
		Type       *string `json:"type,omitempty"`
		PresetKey  *string `json:"preset_key,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	remoteName, err := validateRemoteName(req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	remoteType, err := validateRemoteConfig(remoteName, req.ConfigData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Type != nil && strings.TrimSpace(*req.Type) != "" && strings.TrimSpace(*req.Type) != remoteType {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type does not match config_data"})
		return
	}

	if req.PresetKey != nil && strings.TrimSpace(*req.PresetKey) != "" {
		if err := validateRemotePresetConfig(strings.TrimSpace(*req.PresetKey), remoteName, req.ConfigData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	// Encrypt config data
	encryptedConfig, err := h.cryptoService.Encrypt(req.ConfigData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt config"})
		return
	}

	remoteModel := models.NewRemoteModel(h.db)
	if err := remoteModel.Update(c.Request.Context(), id, remoteName, encryptedConfig, &remoteType); err != nil {
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

// GetExecutionsStats returns execution statistics for the executions page.
func (h *Handler) GetExecutionsStats(c *gin.Context) {
	monitor := services.NewExecutionMonitor(h.db)
	raw, err := monitor.GetExecutionStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get execution stats"})
		return
	}

	pending, _ := raw["pending"].(int)
	running, _ := raw["running"].(int)
	success, _ := raw["success"].(int)
	failed, _ := raw["failed"].(int)

	successRate24h := float64(0)
	if v, ok := raw["success_rate_24h"].(float64); ok {
		successRate24h = v
	}

	avgDurationSeconds := float64(0)
	if v, ok := raw["avg_duration_seconds"].(float64); ok {
		avgDurationSeconds = v
	}

	c.JSON(http.StatusOK, gin.H{
		"total":                pending + running + success + failed,
		"running":              running,
		"success":              success,
		"failed":               failed,
		"success_rate_24h":     successRate24h,
		"avg_duration_seconds": avgDurationSeconds,
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
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Execution not found"})
			return
		}
		log.Printf("[GetExecutionDetail] Failed to load execution %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load execution"})
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
	execution, err := executionModel.Create(c.Request.Context(), taskID, agentID, models.TriggerModeManual)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to trigger execution"})
		return
	}

	// Nudge connected agents to heartbeat immediately so manual triggers dispatch with low latency.
	if h.wsService != nil && h.wsService.IsConnected(agentID) {
		_ = h.wsService.SendJSON(agentID, WSMessage{Type: WSMessageTypeHubPing})
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
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt remote config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                    remote.ID,
		"name":                  remote.Name,
		"type":                  remote.Type,
		"config_data":           decryptedConfig,
		"last_test_at":          remote.LastTestAt,
		"last_test_success":     remote.LastTestSuccess,
		"last_test_message":     remote.LastTestMessage,
		"last_test_error":       remote.LastTestError,
		"last_test_duration_ms": remote.LastTestDuration,
		"created_at":            remote.CreatedAt,
		"updated_at":            remote.UpdatedAt,
	})
}

// TestRemote tests a remote connection via local Agent
func (h *Handler) TestRemote(c *gin.Context) {
	remoteID := c.Param("id")

	remoteModel := models.NewRemoteModel(h.db)
	remote, err := remoteModel.GetByID(c.Request.Context(), uuid.MustParse(remoteID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Remote not found"})
		return
	}

	var req struct {
		TestPath string `json:"test_path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Decrypt the config
	decryptedConfig, err := h.cryptoService.Decrypt(remote.ConfigData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrypt remote config"})
		return
	}

	// Test connection via local Agent
	result, err := h.rcloneService.TestConnection(c.Request.Context(), remote.Name, decryptedConfig, req.TestPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to test connection: " + err.Error()})
		return
	}

	testedAt := time.Now()
	var (
		messagePtr  *string
		errorPtr    *string
		durationPtr *int64
	)
	if strings.TrimSpace(result.Message) != "" {
		msg := result.Message
		messagePtr = &msg
	}
	if strings.TrimSpace(result.Error) != "" {
		errText := result.Error
		errorPtr = &errText
	}
	duration := result.DurationMs
	durationPtr = &duration

	if err := remoteModel.UpdateLastTestResult(c.Request.Context(), remote.ID, testedAt, result.Success, messagePtr, errorPtr, durationPtr); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist remote test result"})
		return
	}

	status := http.StatusOK
	if result.Message == "Failed to connect to local Agent" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"success":     result.Success,
		"message":     result.Message,
		"remote_id":   remoteID,
		"remote_name": remote.Name,
		"duration_ms": result.DurationMs,
		"output":      result.Output,
		"error":       result.Error,
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
		"status":       "cancelled",
		"timestamp":    time.Now().Format(time.RFC3339),
	})

	// If the agent is connected over WebSocket, signal cancellation immediately.
	if h.wsService != nil && h.wsService.IsConnected(execution.AgentID) {
		actionsData, _ := json.Marshal(HeartbeatResponse{
			Actions: []HeartbeatAction{{
				Action:      "CANCEL_TASK",
				Type:        "CANCEL_TASK",
				ExecutionID: execution.ID.String(),
			}},
		})
		_ = h.wsService.SendJSON(execution.AgentID, WSMessage{
			Type: WSMessageTypeHubActions,
			Data: actionsData,
		})
	}

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
			"total":   len(agents),
			"online":  onlineAgents,
			"offline": len(agents) - onlineAgents,
		},
		"tasks": gin.H{
			"total":    len(tasks),
			"active":   activeTasks,
			"inactive": len(tasks) - activeTasks,
		},
		"executions": gin.H{
			"total":        len(executions),
			"success":      successCount,
			"failed":       failedCount,
			"running":      runningCount,
			"success_rate": successRate,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// AdminLogout handles admin logout
func (h *Handler) AdminLogout(c *gin.Context) {
	// Get user ID from context (set by middleware)
	userID, exists := c.Get("user_id")
	userModel := models.NewUserModel(h.db)

	// Revoke session if present
	if sessionVal, hasSession := c.Get("session_id"); hasSession {
		if sessionID, ok := sessionVal.(uuid.UUID); ok {
			if err := userModel.DeleteSession(c.Request.Context(), sessionID); err != nil {
				log.Printf("[Logout] Failed to delete session %s: %v", sessionID, err)
			}
		}
	} else if token, err := extractBearerToken(c); err == nil {
		if session, err := userModel.GetSessionByToken(c.Request.Context(), h.authService.HashToken(token)); err == nil {
			_ = userModel.DeleteSession(c.Request.Context(), session.ID)
		}
	}

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
		"timestamp":  time.Now().Format(time.RFC3339),
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
		"data":      []map[string]interface{}{},
		"range":     timeRange,
		"days":      days,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// DownloadAgentScript provides the agent installation script
func (h *Handler) DownloadAgentScript(c *gin.Context) {
	scriptPath := "./static/scripts/install_agent.sh"
	scriptData, err := os.ReadFile(scriptPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Installation script not found"})
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Data(http.StatusOK, "text/plain", scriptData)
}

// DownloadAgent provides agent binary download
func (h *Handler) DownloadAgent(c *gin.Context) {
	// 获取平台参数
	platform := c.Query("platform")
	arch := c.Query("arch")

	// 如果没有指定平台，尝试从User-Agent检测
	if platform == "" || arch == "" {
		userAgent := c.GetHeader("User-Agent")
		if strings.Contains(userAgent, "Windows") {
			platform = "windows"
			arch = "amd64"
		} else if strings.Contains(userAgent, "Darwin") {
			platform = "darwin"
			if strings.Contains(userAgent, "arm64") || strings.Contains(userAgent, "Apple") {
				arch = "arm64"
			} else {
				arch = "amd64"
			}
		} else {
			// 默认Linux
			platform = "linux"
			arch = "amd64"
		}
	}

	// 构建文件名
	if strings.Contains(platform, "/") || strings.Contains(platform, "\\") || strings.Contains(platform, "..") ||
		strings.Contains(arch, "/") || strings.Contains(arch, "\\") || strings.Contains(arch, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid platform or architecture"})
		return
	}

	validMatrix := map[string][]string{
		"linux":   {"amd64", "arm64", "arm", "386"},
		"windows": {"amd64", "arm64", "386"},
		"darwin":  {"amd64", "arm64"},
	}

	validArch := false
	allowedArchs, ok := validMatrix[platform]
	if ok {
		for _, allowed := range allowedArchs {
			if arch == allowed {
				validArch = true
				break
			}
		}
	}

	if !ok || !validArch {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported platform or architecture"})
		return
	}

	var fileName string
	if platform == "windows" {
		fileName = fmt.Sprintf("rclone-backup-agent-%s-%s.exe", platform, arch)
	} else {
		fileName = fmt.Sprintf("rclone-backup-agent-%s-%s", platform, arch)
	}

	// 设置二进制文件下载的HTTP头
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Transfer-Encoding", "binary")

	// 读取实际的二进制文件
	binaryPath := fmt.Sprintf("./static/binaries/%s", fileName)
	fileData, err := os.ReadFile(binaryPath)
	if err != nil {
		// 如果指定平台的文件不存在，尝试默认文件
		defaultPath := "./static/binaries/rclone-backup-agent"
		fileData, err = os.ReadFile(defaultPath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":               "Agent binary not found",
				"message":             fmt.Sprintf("Binary file for %s/%s not available", platform, arch),
				"available_platforms": []string{"linux/amd64", "linux/arm64", "linux/arm", "darwin/amd64", "darwin/arm64", "windows/amd64"},
			})
			return
		}
		// 使用默认文件名
		fileName = "rclone-backup-agent"
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	}

	// 返回二进制文件
	c.Data(http.StatusOK, "application/octet-stream", fileData)
}
