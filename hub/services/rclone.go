package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// RcloneService handles rclone operations via local Agent
type RcloneService struct {
	localAgentURL string
	localToken    string
	httpClient    *http.Client
}

// TestConnectionResult contains the result of a remote connection test
type TestConnectionResult struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	DurationMs int64  `json:"duration_ms"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

// NewRcloneService creates a new RcloneService
// localAgentURL should be set from LOCAL_AGENT_URL environment variable
func NewRcloneService() *RcloneService {
	localAgentURL := os.Getenv("LOCAL_AGENT_URL")
	if localAgentURL == "" {
		localAgentURL = "http://localhost:9092" // Default local agent API port
	}

	localToken := os.Getenv("LOCAL_AGENT_TOKEN")
	if localToken == "" {
		localToken = os.Getenv("AGENT_API_TOKEN")
	}

	return &RcloneService{
		localAgentURL: localAgentURL,
		localToken:    localToken,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Allow time for remote test
		},
	}
}

// TestConnection tests an rclone remote connection via the local Agent
func (s *RcloneService) TestConnection(ctx context.Context, remoteName string, configData string, testPath string) (*TestConnectionResult, error) {
	// Prepare request body
	reqBody := map[string]string{
		"remote_name": remoteName,
		"config_data": configData,
	}
	if testPath = strings.TrimSpace(testPath); testPath != "" {
		reqBody["test_path"] = testPath
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make request to local Agent
	url := s.localAgentURL + "/api/test-remote"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.localToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.localToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "Failed to connect to local Agent",
			Error:   fmt.Sprintf("Local Agent unavailable at %s: %v", s.localAgentURL, err),
		}, nil
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &TestConnectionResult{
			Success: false,
			Message: "Local Agent returned error",
			Error:   string(body),
		}, nil
	}

	// Parse result
	var result TestConnectionResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// IsAvailable checks if the local Agent API is available
func (s *RcloneService) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", s.localAgentURL+"/api/health", nil)
	if err != nil {
		return false
	}
	if s.localToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.localToken)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// GetAgentURL returns the local agent URL
func (s *RcloneService) GetAgentURL() string {
	return s.localAgentURL
}
