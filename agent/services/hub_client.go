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
	collector  *SystemCollector
	version    string
}

// HeartbeatRequest represents a heartbeat request
type HeartbeatRequest struct {
	AgentID    string      `json:"agent_id"`
	Status     string      `json:"status"`
	Tasks      interface{} `json:"tasks,omitempty"`
	SystemInfo SystemInfo  `json:"system_info"`
	Timestamp  time.Time   `json:"timestamp"`
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

	BackupMode          string `json:"backup_mode,omitempty"`
	ArchiveFormat       string `json:"archive_format,omitempty"`
	EncryptionEnabled   bool   `json:"encryption_enabled"`
	EncryptionPassword  string `json:"encryption_password,omitempty"`
	EncryptionPassword2 string `json:"encryption_password2,omitempty"`
}

type FSListResult struct {
	RequestID string        `json:"request_id"`
	Path      string        `json:"path"`
	Parent    string        `json:"parent,omitempty"`
	Entries   []FSListEntry `json:"entries"`
	Error     string        `json:"error,omitempty"`
}

// NewHubClient creates a new hub client
func NewHubClient(baseURL, agentID, apiKey, version string) *HubClient {
	return &HubClient{
		baseURL: baseURL,
		agentID: agentID,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		collector: NewSystemCollector(),
		version:   version,
	}
}

// SetCredentials updates the agent ID and API key for the client
func (c *HubClient) SetCredentials(agentID, apiKey string) {
	c.agentID = agentID
	c.apiKey = apiKey
}

// Register registers the agent with the hub
func (c *HubClient) Register(ctx context.Context, token, name string, isLocal bool) (string, string, error) {
	req := map[string]interface{}{
		"token":    token,
		"name":     name,
		"is_local": isLocal,
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

func (c *HubClient) BuildHeartbeat(status string, tasks interface{}) HeartbeatRequest {
	return HeartbeatRequest{
		AgentID:    c.agentID,
		Status:     status,
		Tasks:      tasks,
		SystemInfo: c.collector.Collect(c.version),
		Timestamp:  time.Now(),
	}
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
	if c.apiKey != "" && c.agentID != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s:%s", c.agentID, c.apiKey))
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
