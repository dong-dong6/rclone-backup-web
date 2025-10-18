package rclone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for rclone rcd API
type Client struct {
	endpoint   string
	httpClient *http.Client
	username   string
	password   string
}

// NewClient creates a new rclone rcd client
func NewClient(endpoint, username, password string) *Client {
	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		username: username,
		password: password,
	}
}

// JobStatus represents the status of a rclone job
type JobStatus struct {
	ID         int       `json:"id"`
	Group      string    `json:"group"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime,omitempty"`
	Duration   float64   `json:"duration"`
	Speed      float64   `json:"speed"`
	Bytes      int64     `json:"bytes"`
	Checks     int       `json:"checks"`
	Transfers  int       `json:"transfers"`
	Errors     int       `json:"errors"`
	Finished   bool      `json:"finished"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
	Output     []string  `json:"output,omitempty"`
}

// CoreStats represents rclone core statistics
type CoreStats struct {
	Speed       float64 `json:"speed"`
	Bytes       int64   `json:"bytes"`
	Errors      int     `json:"errors"`
	Checks      int     `json:"checks"`
	Transfers   int     `json:"transfers"`
	Deletes     int     `json:"deletes"`
	ElapsedTime float64 `json:"elapsedTime"`
}

// request makes an HTTP request to rclone rcd
func (c *Client) request(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	
	// Add basic auth if configured
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("rclone rcd returned error %d: %s", resp.StatusCode, body)
	}

	return resp, nil
}

// Sync performs a sync operation
func (c *Client) Sync(ctx context.Context, source, destination string, args []string) (*JobStatus, error) {
	params := map[string]interface{}{
		"srcFs": source,
		"dstFs": destination,
		"_async": true, // Run asynchronously
	}

	// Add additional arguments
	for i := 0; i < len(args); i++ {
		if args[i] == "--dry-run" {
			params["dryRun"] = true
		} else if args[i] == "--checksum" {
			params["checkSum"] = true
		} else if args[i] == "--fast-list" {
			params["fastList"] = true
		} else if args[i] == "--transfers" && i+1 < len(args) {
			params["transfers"] = args[i+1]
			i++
		}
	}

	resp, err := c.request(ctx, "POST", "/sync/sync", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		JobID int `json:"jobid"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Poll for job status
	return c.waitForJob(ctx, result.JobID)
}

// Copy performs a copy operation
func (c *Client) Copy(ctx context.Context, source, destination string, args []string) (*JobStatus, error) {
	params := map[string]interface{}{
		"srcFs": source,
		"dstFs": destination,
		"_async": true,
	}

	resp, err := c.request(ctx, "POST", "/sync/copy", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		JobID int `json:"jobid"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.waitForJob(ctx, result.JobID)
}

// Move performs a move operation
func (c *Client) Move(ctx context.Context, source, destination string, args []string) (*JobStatus, error) {
	params := map[string]interface{}{
		"srcFs": source,
		"dstFs": destination,
		"_async": true,
	}

	resp, err := c.request(ctx, "POST", "/sync/move", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		JobID int `json:"jobid"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.waitForJob(ctx, result.JobID)
}

// GetJobStatus gets the status of a job
func (c *Client) GetJobStatus(ctx context.Context, jobID int) (*JobStatus, error) {
	params := map[string]interface{}{
		"jobid": jobID,
	}

	resp, err := c.request(ctx, "POST", "/job/status", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var status JobStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode job status: %w", err)
	}

	return &status, nil
}

// waitForJob polls for job completion
func (c *Client) waitForJob(ctx context.Context, jobID int) (*JobStatus, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := c.GetJobStatus(ctx, jobID)
			if err != nil {
				return nil, err
			}
			
			if status.Finished {
				return status, nil
			}
		}
	}
}

// GetStats gets current transfer statistics
func (c *Client) GetStats(ctx context.Context) (*CoreStats, error) {
	resp, err := c.request(ctx, "POST", "/core/stats", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats CoreStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	return &stats, nil
}

// ListRemotes lists configured remotes
func (c *Client) ListRemotes(ctx context.Context) ([]string, error) {
	resp, err := c.request(ctx, "POST", "/config/listremotes", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Remotes []string `json:"remotes"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode remotes: %w", err)
	}

	return result.Remotes, nil
}

// CreateRemote creates a new remote configuration
func (c *Client) CreateRemote(ctx context.Context, name string, config map[string]string) error {
	params := map[string]interface{}{
		"name":       name,
		"parameters": config,
	}

	resp, err := c.request(ctx, "POST", "/config/create", params)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// DeleteRemote deletes a remote configuration
func (c *Client) DeleteRemote(ctx context.Context, name string) error {
	params := map[string]interface{}{
		"name": name,
	}

	resp, err := c.request(ctx, "POST", "/config/delete", params)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// TestRemote tests a remote connection
func (c *Client) TestRemote(ctx context.Context, remotePath string) error {
	params := map[string]interface{}{
		"fs": remotePath,
	}

	resp, err := c.request(ctx, "POST", "/operations/about", params)
	if err != nil {
		return fmt.Errorf("remote test failed: %w", err)
	}
	resp.Body.Close()

	return nil
}

// Version gets rclone version information
func (c *Client) Version(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.request(ctx, "POST", "/core/version", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var version map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return nil, fmt.Errorf("failed to decode version: %w", err)
	}

	return version, nil
}

// Quit stops the rclone rcd server
func (c *Client) Quit(ctx context.Context) error {
	resp, err := c.request(ctx, "POST", "/core/quit", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}