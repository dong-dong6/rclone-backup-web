package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// AgentAPIServer provides HTTP API for the agent
type AgentAPIServer struct {
	port       int
	bindAddr   string
	authToken  string
	workDir    string
	rclonePath string
	server     *http.Server
}

// TestRemoteRequest represents a request to test a remote connection
type TestRemoteRequest struct {
	RemoteName string `json:"remote_name"`
	ConfigData string `json:"config_data"` // Decrypted rclone config content
}

// TestRemoteResponse represents the response from testing a remote
type TestRemoteResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	DurationMs int64  `json:"duration_ms"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

// NewAgentAPIServer creates a new API server for the agent
func NewAgentAPIServer(port int, workDir string, rclonePath string, bindAddr string, authToken string) *AgentAPIServer {
	return &AgentAPIServer{
		port:       port,
		bindAddr:   bindAddr,
		authToken:  authToken,
		workDir:    workDir,
		rclonePath: rclonePath,
	}
}

// Start starts the HTTP API server
func (s *AgentAPIServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Register endpoints
	mux.HandleFunc("/api/test-remote", s.handleTestRemote)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/version", s.handleVersion)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.bindAddr, s.port),
		Handler: mux,
	}

	log.Printf("[AgentAPI] Starting API server on %s:%d", s.bindAddr, s.port)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[AgentAPI] Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(shutdownCtx)
}

// handleTestRemote handles POST /api/test-remote
func (s *AgentAPIServer) handleTestRemote(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(w, r) {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TestRemoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.RemoteName == "" || req.ConfigData == "" {
		s.jsonError(w, "remote_name and config_data are required", http.StatusBadRequest)
		return
	}

	// Perform the test
	result := s.testRemoteConnection(r.Context(), req.RemoteName, req.ConfigData)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// testRemoteConnection tests an rclone remote connection
func (s *AgentAPIServer) testRemoteConnection(ctx context.Context, remoteName, configData string) *TestRemoteResponse {
	startTime := time.Now()

	// Create temporary config file
	configDir := filepath.Join(s.workDir, "test-configs")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return &TestRemoteResponse{
			Success:    false,
			Message:    "Failed to create config directory",
			Error:      err.Error(),
			DurationMs: time.Since(startTime).Milliseconds(),
		}
	}

	configPath := filepath.Join(configDir, fmt.Sprintf("test-%d.conf", time.Now().UnixNano()))
	if err := os.WriteFile(configPath, []byte(configData), 0600); err != nil {
		return &TestRemoteResponse{
			Success:    false,
			Message:    "Failed to write config file",
			Error:      err.Error(),
			DurationMs: time.Since(startTime).Milliseconds(),
		}
	}
	defer os.Remove(configPath)

	// Run rclone lsd with timeout
	testCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(testCtx, s.rclonePath, "lsd", remoteName+":", "--config", configPath)
	output, err := cmd.CombinedOutput()

	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		if testCtx.Err() == context.DeadlineExceeded {
			return &TestRemoteResponse{
				Success:    false,
				Message:    "Connection test timed out (30s)",
				DurationMs: duration,
			}
		}

		return &TestRemoteResponse{
			Success:    false,
			Message:    "Connection test failed",
			Error:      strings.TrimSpace(string(output)),
			DurationMs: duration,
		}
	}

	return &TestRemoteResponse{
		Success:    true,
		Message:    "Remote connection successful",
		Output:     strings.TrimSpace(string(output)),
		DurationMs: duration,
	}
}

// handleHealth handles GET /api/health
func (s *AgentAPIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(w, r) {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Unix(),
	})
}

// handleVersion handles GET /api/version
func (s *AgentAPIServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(w, r) {
		return
	}

	// Get rclone version
	cmd := exec.Command(s.rclonePath, "version")
	output, err := cmd.Output()
	rcloneVersion := "unknown"
	if err == nil {
		lines := strings.Split(string(output), "\n")
		if len(lines) > 0 {
			rcloneVersion = strings.TrimSpace(lines[0])
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"agent_version":  "1.0.0",
		"rclone_version": rcloneVersion,
	})
}

// jsonError writes a JSON error response
func (s *AgentAPIServer) jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// authorize checks Authorization header if a token is configured
func (s *AgentAPIServer) authorize(w http.ResponseWriter, r *http.Request) bool {
	if s.authToken == "" {
		return true
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		s.jsonError(w, "Authorization header missing", http.StatusUnauthorized)
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] != s.authToken {
		s.jsonError(w, "Invalid or unauthorized token", http.StatusUnauthorized)
		return false
	}

	return true
}
