package main

package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rclone-backup-web/agent/rclone"
)

// executeTaskWithSidecar executes a task using rclone sidecar
func (a *Agent) executeTaskWithSidecar(executionID string, task *Task, triggerMode string) {
	log.Printf("Executing task %s via sidecar (execution: %s, trigger: %s)", 
		task.TaskID, executionID, triggerMode)

	a.runningMux.Lock()
	a.isRunningTask = true
	a.runningMux.Unlock()
	defer func() {
		a.runningMux.Lock()
		a.isRunningTask = false
		a.runningMux.Unlock()
	}()

	startTime := time.Now()
	ctx := context.Background()

	// Initialize rclone client
	rcloneClient := rclone.NewClient(
		a.config.RcloneEndpoint,
		os.Getenv("RCLONE_RC_USER"),
		os.Getenv("RCLONE_RC_PASS"),
	)

	// Decode and create remote config
	if task.RcloneConfigB64 != "" {
		configData, err := base64.StdEncoding.DecodeString(task.RcloneConfigB64)
		if err != nil {
			log.Printf("Failed to decode rclone config: %v", err)
			a.reportExecutionResult(executionID, "failed", 
				fmt.Sprintf("Config decode error: %v", err), startTime)
			return
		}

		// Parse config and create remote
		configMap := parseRcloneConfig(string(configData))
		if err := rcloneClient.CreateRemote(ctx, "backup-remote", configMap); err != nil {
			log.Printf("Failed to create remote: %v", err)
			a.reportExecutionResult(executionID, "failed", 
				fmt.Sprintf("Remote creation error: %v", err), startTime)
			return
		}
		defer rcloneClient.DeleteRemote(ctx, "backup-remote")
	}

	// Prepare source and destination paths
	source := task.SourcePath
	destination := fmt.Sprintf("backup-remote:%s", task.DestinationPath)

	// Start log streaming goroutine
	logChan := make(chan string, 100)
	go a.streamLogsFromChannel(executionID, logChan)

	// Start stats monitoring with execution ID for log tracking
	go a.monitorTransferStats(ctx, rcloneClient, executionID, logChan)

	// Execute sync operation
	logChan <- fmt.Sprintf("Starting backup: %s -> %s", source, destination)
	
	jobStatus, err := rcloneClient.Sync(ctx, source, destination, task.RcloneArgs)
	
	if err != nil {
		status := "failed"
		errorMsg := fmt.Sprintf("Sync failed: %v", err)
		logChan <- errorMsg
		close(logChan)
		
		log.Printf("Task execution failed: %v", err)
		a.reportExecutionResult(executionID, status, errorMsg, startTime)
		return
	}

	// Check job results
	status := "success"
	var finalOutput string
	
	if jobStatus.Success {
		finalOutput = fmt.Sprintf(
			"Backup completed successfully\n"+
			"Files transferred: %d\n"+
			"Bytes transferred: %d\n"+
			"Errors: %d\n"+
			"Duration: %.2f seconds\n"+
			"Average speed: %.2f MB/s",
			jobStatus.Transfers,
			jobStatus.Bytes,
			jobStatus.Errors,
			jobStatus.Duration,
			jobStatus.Speed/1024/1024,
		)
		logChan <- "Backup completed successfully"
	} else {
		status = "failed"
		finalOutput = fmt.Sprintf(
			"Backup failed\n"+
			"Error: %s\n"+
			"Files transferred: %d\n"+
			"Errors: %d",
			jobStatus.Error,
			jobStatus.Transfers,
			jobStatus.Errors,
		)
		logChan <- fmt.Sprintf("Backup failed: %s", jobStatus.Error)
	}
	
	close(logChan)
	
	// Add output lines if available
	if len(jobStatus.Output) > 0 {
		finalOutput += "\n\nDetailed output:\n"
		for _, line := range jobStatus.Output {
			finalOutput += line + "\n"
		}
	}

	log.Printf("Task %s completed with status: %s", task.TaskID, status)
	a.reportExecutionResult(executionID, status, finalOutput, startTime)
}

// monitorTransferStats monitors and reports transfer statistics
func (a *Agent) monitorTransferStats(ctx context.Context, client *rclone.Client, executionID string, logChan chan<- string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	lastLogBatch := time.Now()
	logBatch := []map[string]string{}

	for {
		select {
		case <-ctx.Done():
			// Send any remaining logs
			if len(logBatch) > 0 {
				a.sendLogBatch(executionID, logBatch)
			}
			return
		case <-ticker.C:
			stats, err := client.GetStats(ctx)
			if err != nil {
				continue
			}

			if stats.Transfers > 0 || stats.Bytes > 0 {
				logMsg := fmt.Sprintf(
					"Progress: %d files, %.2f MB transferred, %.2f MB/s, %d errors",
					stats.Transfers,
					float64(stats.Bytes)/1024/1024,
					stats.Speed/1024/1024,
					stats.Errors,
				)
				logChan <- logMsg
				
				// Add to batch for hub reporting
				logBatch = append(logBatch, map[string]string{
					"timestamp": time.Now().Format(time.RFC3339),
					"message":   logMsg,
				})
				
				// Send batch if it's been 10 seconds or we have 10+ logs
				if time.Since(lastLogBatch) > 10*time.Second || len(logBatch) >= 10 {
					a.sendLogBatch(executionID, logBatch)
					logBatch = []map[string]string{}
					lastLogBatch = time.Now()
				}
			}

			// Check for detailed transfer information
			if detailedStats, err := client.GetTransferStats(ctx, "job/"+executionID); err == nil {
				for _, file := range detailedStats.Transferring {
					fileMsg := fmt.Sprintf(
						"Transferring: %s (%.1f%% of %.2f MB @ %.2f MB/s)",
						file.Name,
						file.Percentage,
						float64(file.Size)/1024/1024,
						file.Speed/1024/1024,
					)
					logChan <- fileMsg
				}
			}
		}
	}
}

// streamLogsFromChannel streams logs from a channel to the hub
func (a *Agent) streamLogsFromChannel(executionID string, logChan <-chan string) {
	batch := make([]map[string]string, 0, 10)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case log, ok := <-logChan:
			if !ok {
				// Channel closed, send remaining logs
				if len(batch) > 0 {
					a.sendLogBatch(executionID, batch)
				}
				return
			}
			
			batch = append(batch, map[string]string{
				"timestamp": time.Now().Format(time.RFC3339),
				"message":   log,
			})

			// Send batch if it's full
			if len(batch) >= 10 {
				a.sendLogBatch(executionID, batch)
				batch = make([]map[string]string, 0, 10)
			}

		case <-ticker.C:
			// Send batch periodically
			if len(batch) > 0 {
				a.sendLogBatch(executionID, batch)
				batch = make([]map[string]string, 0, 10)
			}
		}
	}
}

// sendLogBatch sends a batch of logs to the hub
func (a *Agent) sendLogBatch(executionID string, logs []map[string]string) {
	reqBody := map[string]interface{}{
		"logs": logs,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/api/v1/agent/executions/%s/logs", a.config.HubURL, executionID),
		bytes.NewReader(bodyBytes))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

// parseRcloneConfig parses rclone config string into a map
func parseRcloneConfig(config string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(config, "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "[") {
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}
	
	return result
}

// UseSidecar determines whether to use sidecar or direct execution
func (a *Agent) UseSidecar() bool {
	// Check if sidecar endpoint is configured and accessible
	if a.config.RcloneEndpoint == "" {
		return false
	}
	
	// Try to connect to sidecar
	client := rclone.NewClient(
		a.config.RcloneEndpoint,
		os.Getenv("RCLONE_RC_USER"),
		os.Getenv("RCLONE_RC_PASS"),
	)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	_, err := client.Version(ctx)
	return err == nil
}