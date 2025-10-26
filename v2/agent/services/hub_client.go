package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HubClient handles communication with the hub
type HubClient struct {
	baseURL    string
	agentID    string
	apiKey     string
	httpClient *http.Client
}

// HeartbeatRequest represents a heartbeat request
type HeartbeatRequest struct {
	AgentID    string      `json:"agent_id"`
	Status     string      `json:"status"`
	Tasks      interface{} `json:"tasks,omitempty"`
	SystemInfo SystemInfo  `json:"system_info"`
	Timestamp  time.Time   `json:"timestamp"`
}

// SystemInfo contains system information
type SystemInfo struct {
	Hostname     string  `json:"hostname"`
	OS           string  `json:"os"`
	Arch         string  `json:"arch"`
	CPUUsage     float64 `json:"cpu_usage"`
	MemoryUsage  float64 `json:"memory_usage"`
	DiskUsage    float64 `json:"disk_usage"`
	AgentVersion string  `json:"agent_version"`
}

// HeartbeatResponse represents a heartbeat response
type HeartbeatResponse struct {
	Success bool     `json:"success"`
	Actions []Action `json:"actions,omitempty"`
	Message string   `json:"message,omitempty"`
}

// Action represents an action from the hub
type Action struct {
	Type        string          `json:"type"`
	ExecutionID string          `json:"execution_id,omitempty"`
	Task        json.RawMessage `json:"task,omitempty"`
	Config      json.RawMessage `json:"config,omitempty"`
}

// Task represents a task from hub
type Task struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Schedule     string   `json:"schedule"`
	RemoteConfig string   `json:"remote_config"`
	SourcePath   string   `json:"source_path"`
	DestPath     string   `json:"dest_path"`
	RcloneArgs   []string `json:"rclone_args"`
	Enabled      bool     `json:"enabled"`
}

// NewHubClient creates a new hub client
func NewHubClient(baseURL, agentID, apiKey string) *HubClient {
	return &HubClient{
		baseURL: baseURL,
		agentID: agentID,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetCredentials updates the agent ID and API key for the client
func (c *HubClient) SetCredentials(agentID, apiKey string) {
	c.agentID = agentID
	c.apiKey = apiKey
}

// Register registers the agent with the hub
func (c *HubClient) Register(ctx context.Context, token, name string) (string, string, error) {
	req := map[string]string{
		"token": token,
		"name":  name,
	}
	
	resp := struct {
		AgentID string `json:"agent_id"`
		APIKey  string `json:"api_key"`
	}{}
	
	if err := c.doRequest(ctx, "POST", "/api/v1/agent/register", req, &resp); err != nil {
		return "", "", err
	}
	
	return resp.AgentID, resp.APIKey, nil
}

// SendHeartbeat sends a heartbeat to the hub
func (c *HubClient) SendHeartbeat(ctx context.Context, status string, tasks interface{}) (*HeartbeatResponse, error) {
	req := HeartbeatRequest{
		AgentID:    c.agentID,
		Status:     status,
		Tasks:      tasks,
		SystemInfo: c.getSystemInfo(),
		Timestamp:  time.Now(),
	}
	
	var resp HeartbeatResponse
	if err := c.doRequest(ctx, "POST", "/api/v1/agent/heartbeat", req, &resp); err != nil {
		return nil, err
	}
	
	return &resp, nil
}

// UpdateExecutionStatus updates the status of a task execution
func (c *HubClient) UpdateExecutionStatus(executionID, status string, err error) error {
	req := map[string]interface{}{
		"execution_id": executionID,
		"status":       status,
		"timestamp":    time.Now(),
	}
	
	if err != nil {
		req["error"] = err.Error()
	}
	
	return c.doRequest(context.Background(), "POST", "/api/v1/agent/execution/status", req, nil)
}

// SendLogs sends execution logs to the hub
func (c *HubClient) SendLogs(executionID string, logs []string) error {
	req := map[string]interface{}{
		"execution_id": executionID,
		"logs":         logs,
		"timestamp":    time.Now(),
	}
	
	return c.doRequest(context.Background(), "POST", "/api/v1/agent/execution/logs", req, nil)
}

// GetTasks retrieves tasks assigned to this agent
func (c *HubClient) GetTasks(ctx context.Context) ([]Task, error) {
	var tasks []Task
	if err := c.doRequest(ctx, "GET", "/api/v1/agent/tasks", nil, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// doRequest performs an HTTP request to the hub
func (c *HubClient) doRequest(ctx context.Context, method, path string, body interface{}, response interface{}) error {
	url := c.baseURL + path
	
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewBuffer(data)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("hub returned %d: %s", resp.StatusCode, body)
	}
	
	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	
	return nil
}

// getSystemInfo gathers system information
func (c *HubClient) getSystemInfo() SystemInfo {
	// This is a simplified version - would need proper implementation
	return SystemInfo{
		Hostname:     getHostname(),
		OS:           getOS(),
		Arch:         getArch(),
		CPUUsage:     getCPUUsage(),
		MemoryUsage:  getMemoryUsage(),
		DiskUsage:    getDiskUsage(),
		AgentVersion: getAgentVersion(),
	}
}

// Helper functions (these would need proper implementation)
func getHostname() string {
	// Implementation
	return "agent-host"
}

func getOS() string {
	// Implementation
	return "linux"
}

func getArch() string {
	// Implementation
	return "amd64"
}

func getCPUUsage() float64 {
	// Implementation
	return 0.0
}

func getMemoryUsage() float64 {
	// Implementation
	return 0.0
}

func getDiskUsage() float64 {
	// Implementation
	return 0.0
}

func getAgentVersion() string {
	// Implementation
	return "1.0.0"
}