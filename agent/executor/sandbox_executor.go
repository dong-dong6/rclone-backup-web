package executor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rclone-backup-web/agent/rclone_manager"
)

// TaskExecutor handles isolated task execution
type TaskExecutor struct {
	rcloneManager *rclone_manager.Manager
	configManager *rclone_manager.ConfigManager
	workDir       string
	maxConcurrent int
	activeTasks   sync.Map
	taskSemaphore chan struct{}
	logHook       func(executionID string, line string)
}

// TaskInfo contains task execution information
type TaskInfo struct {
	ID                  string             `json:"id"`
	ExecutionID         string             `json:"execution_id"`
	TaskID              string             `json:"task_id"`
	TaskName            string             `json:"task_name"`
	RemoteConfig        string             `json:"remote_config"`
	SourcePath          string             `json:"source_path"`
	DestPath            string             `json:"dest_path"`
	RcloneArgs          []string           `json:"rclone_args"`
	BackupMode          string             `json:"backup_mode,omitempty"`
	ArchiveFormat       string             `json:"archive_format,omitempty"`
	EncryptionEnabled   bool               `json:"encryption_enabled,omitempty"`
	EncryptionPassword  string             `json:"-"`
	EncryptionPassword2 string             `json:"-"`
	MaxRetention        int                `json:"max_retention,omitempty"`
	StartedAt           time.Time          `json:"started_at"`
	Status              string             `json:"status"`
	Progress            *TransferProgress  `json:"progress,omitempty"`
	Context             context.Context    `json:"-"`
	Cancel              context.CancelFunc `json:"-"`
}

// TransferProgress tracks transfer statistics
type TransferProgress struct {
	BytesTransferred int64     `json:"bytes_transferred"`
	BytesTotal       int64     `json:"bytes_total"`
	FilesTransferred int       `json:"files_transferred"`
	FilesTotal       int       `json:"files_total"`
	Speed            string    `json:"speed"`
	Percentage       float64   `json:"percentage"`
	ETA              string    `json:"eta"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NewTaskExecutor creates a new task executor
func NewTaskExecutor(workDir string, maxConcurrent int) (*TaskExecutor, error) {
	// Initialize rclone manager
	rcloneManager := rclone_manager.NewManager(workDir)
	if _, err := rcloneManager.EnsureRcloneExists(); err != nil {
		return nil, fmt.Errorf("failed to ensure rclone: %w", err)
	}

	// Initialize config manager
	configManager := rclone_manager.NewConfigManager(workDir)

	return &TaskExecutor{
		rcloneManager: rcloneManager,
		configManager: configManager,
		workDir:       workDir,
		maxConcurrent: maxConcurrent,
		taskSemaphore: make(chan struct{}, maxConcurrent),
	}, nil
}

// SetLogHook registers a callback invoked for each rclone output line (stdout/stderr).
// The hook should be non-blocking; it runs in the rclone output processing goroutines.
func (te *TaskExecutor) SetLogHook(hook func(executionID string, line string)) {
	te.logHook = hook
}

// ExecuteTask executes a task in an isolated environment
func (te *TaskExecutor) ExecuteTask(ctx context.Context, task *TaskInfo) error {
	// Acquire semaphore to limit concurrent tasks
	select {
	case te.taskSemaphore <- struct{}{}:
		defer func() { <-te.taskSemaphore }()
	case <-ctx.Done():
		return ctx.Err()
	}

	// Create task context with cancellation
	taskCtx, cancel := context.WithCancel(ctx)
	task.Context = taskCtx
	task.Cancel = cancel
	defer cancel()

	// Store active task
	te.activeTasks.Store(task.ExecutionID, task)
	defer te.activeTasks.Delete(task.ExecutionID)

	// Update status
	task.Status = "preparing"
	task.StartedAt = time.Now()

	log.Printf("[Executor] Starting task %s (execution: %s)", task.TaskID, task.ExecutionID)

	// Create isolated work directory
	taskWorkDir := filepath.Join(te.workDir, "tasks", task.ExecutionID)
	if err := os.MkdirAll(taskWorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}
	defer te.cleanupTaskWorkDir(taskWorkDir)

	// Pre-flight: create archive if requested.
	backupMode := strings.ToLower(strings.TrimSpace(task.BackupMode))
	if backupMode == "" {
		backupMode = "sync"
	}
	if backupMode == "archive" {
		archiveFormat := strings.ToLower(strings.TrimSpace(task.ArchiveFormat))
		if archiveFormat == "" {
			archiveFormat = "tar.gz"
		}
		archiveName := buildArchiveName(task, archiveFormat, time.Now())
		archivePath := filepath.Join(taskWorkDir, archiveName)
		if te.logHook != nil {
			te.logHook(task.ExecutionID, fmt.Sprintf("[archive] creating %s from %s", archiveName, task.SourcePath))
		}
		if err := CreateArchive(taskCtx, task.SourcePath, archivePath, archiveFormat); err != nil {
			return fmt.Errorf("failed to create archive: %w", err)
		}
		if te.logHook != nil {
			te.logHook(task.ExecutionID, fmt.Sprintf("[archive] created %s", archiveName))
		}
		task.SourcePath = archivePath
	}

	// Prepare rclone config (optionally adding a per-task crypt remote).
	configContent := task.RemoteConfig
	if task.EncryptionEnabled {
		if strings.TrimSpace(task.EncryptionPassword) == "" || strings.TrimSpace(task.EncryptionPassword2) == "" {
			return fmt.Errorf("encryption enabled but missing password(s)")
		}

		baseConfig, err := decodeMaybeBase64(task.RemoteConfig)
		if err != nil {
			return fmt.Errorf("failed to decode remote config: %w", err)
		}

		obscuredPassword, err := te.obscureSecret(taskCtx, task.EncryptionPassword)
		if err != nil {
			return err
		}
		obscuredPassword2, err := te.obscureSecret(taskCtx, task.EncryptionPassword2)
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		buf.WriteString(string(baseConfig))
		if !strings.HasSuffix(buf.String(), "\n") {
			buf.WriteString("\n")
		}
		buf.WriteString("\n[crypt]\n")
		buf.WriteString("type = crypt\n")
		buf.WriteString("remote = remote:\n")
		buf.WriteString("filename_encryption = standard\n")
		buf.WriteString("directory_name_encryption = true\n")
		buf.WriteString("password = ")
		buf.WriteString(obscuredPassword)
		buf.WriteString("\npassword2 = ")
		buf.WriteString(obscuredPassword2)
		buf.WriteString("\n")

		configContent = buf.String()
	}

	// Create temporary rclone config
	configPath, err := te.configManager.CreateTempConfig(task.ExecutionID, configContent)
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}
	defer te.configManager.CleanupConfig(task.ExecutionID)

	// Update status
	task.Status = "running"

	// Execute rclone command
	if err := te.runRclone(taskCtx, task, configPath, taskWorkDir); err != nil {
		task.Status = "failed"
		return fmt.Errorf("rclone execution failed: %w", err)
	}

	// Post-flight: enforce max retention for archive mode.
	if backupMode == "archive" && task.MaxRetention > 0 {
		if err := te.enforceArchiveRetention(taskCtx, task, configPath, taskWorkDir); err != nil {
			if te.logHook != nil {
				te.logHook(task.ExecutionID, fmt.Sprintf("[retention] warning: %v", err))
			}
		}
	}

	task.Status = "completed"
	log.Printf("[Executor] Task %s completed successfully", task.ExecutionID)

	return nil
}

// runRclone executes the rclone command
func (te *TaskExecutor) runRclone(ctx context.Context, task *TaskInfo, configPath, workDir string) error {
	// Build rclone command
	rclonePath := te.rcloneManager.GetExecutablePath()

	// Base command: rclone sync/copy source dest
	operation := "sync"
	if strings.EqualFold(strings.TrimSpace(task.BackupMode), "archive") {
		operation = "copy"
	}
	args := []string{
		operation,
		task.SourcePath,
		task.DestPath,
		"--config", configPath,
		"--progress",
		"--stats", "1s",
		"--stats-one-line",
		"--log-level", "INFO",
		"--use-json-log",
	}

	// Add user-specified arguments
	args = append(args, expandRcloneArgs(task.RcloneArgs)...)

	log.Printf("[Executor] Running command: %s %s", rclonePath, strings.Join(args, " "))
	if te.logHook != nil {
		if summary := summarizeConfigForLog(configPath); summary != "" {
			te.logHook(task.ExecutionID, summary)
		}
		te.logHook(task.ExecutionID, formatCommandForLog(rclonePath, args))
	}

	// Create command
	cmd := exec.CommandContext(ctx, rclonePath, args...)
	cmd.Dir = workDir

	// Set environment variables for isolation
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("RCLONE_CONFIG=%s", configPath),
		"RCLONE_NO_CHECK_CERTIFICATE=false",
		fmt.Sprintf("TMPDIR=%s", workDir),
	)

	// Capture stdout for progress parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Capture stderr for error messages
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start rclone: %w", err)
	}

	// Process output in goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Process stdout (progress)
	go func() {
		defer wg.Done()
		te.processRcloneOutput(task, stdout, false)
	}()

	// Process stderr (logs)
	go func() {
		defer wg.Done()
		te.processRcloneOutput(task, stderr, true)
	}()

	// Wait for output processing
	wg.Wait()

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("task cancelled: %w", ctx.Err())
		}
		return fmt.Errorf("rclone command failed: %w", err)
	}

	return nil
}

func decodeMaybeBase64(input string) ([]byte, error) {
	if decoded, err := base64.StdEncoding.DecodeString(input); err == nil {
		return decoded, nil
	}
	return []byte(input), nil
}

func (te *TaskExecutor) obscureSecret(ctx context.Context, secret string) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", fmt.Errorf("secret is empty")
	}

	rclonePath := te.rcloneManager.GetExecutablePath()
	cmd := exec.CommandContext(ctx, rclonePath, "obscure", "-")
	cmd.Stdin = strings.NewReader(secret + "\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("rclone obscure failed: %s", message)
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return "", fmt.Errorf("rclone obscure returned empty output")
	}
	return out, nil
}

// processRcloneOutput processes rclone output for progress and logs
func (te *TaskExecutor) processRcloneOutput(task *TaskInfo, reader io.Reader, isError bool) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	scanner.Split(scanLinesCRLF)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if te.logHook != nil {
			te.logHook(task.ExecutionID, line)
		}

		// Try to parse as JSON log
		var jsonLog map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonLog); err == nil {
			te.parseRcloneJSON(task, jsonLog)
		} else {
			// Parse plain text progress
			te.parseRclonePlainText(task, line)
		}

		// Log to console
		if isError {
			log.Printf("[Executor][ERROR] %s", line)
		} else {
			log.Printf("[Executor] %s", line)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[Executor] rclone output reader error (execution=%s): %v", task.ExecutionID, err)
		// Best-effort drain to avoid blocking the rclone process on a full pipe.
		_, _ = io.Copy(io.Discard, reader)
	}
}

func scanLinesCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' || data[i] == '\r' {
			end := i
			advance = i + 1
			if data[i] == '\r' && i+1 < len(data) && data[i+1] == '\n' {
				advance = i + 2
			}
			return advance, data[:end], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// parseRcloneJSON parses JSON formatted rclone output
func (te *TaskExecutor) parseRcloneJSON(task *TaskInfo, jsonLog map[string]interface{}) {
	// Look for stats in JSON
	if stats, ok := jsonLog["stats"].(map[string]interface{}); ok {
		progress := &TransferProgress{
			UpdatedAt: time.Now(),
		}

		if bytes, ok := stats["bytes"].(float64); ok {
			progress.BytesTransferred = int64(bytes)
		}
		if totalBytes, ok := stats["totalBytes"].(float64); ok {
			progress.BytesTotal = int64(totalBytes)
		}
		if transfers, ok := stats["transfers"].(float64); ok {
			progress.FilesTransferred = int(transfers)
		}
		if totalTransfers, ok := stats["totalTransfers"].(float64); ok {
			progress.FilesTotal = int(totalTransfers)
		}
		if speed, ok := stats["speed"].(float64); ok {
			progress.Speed = formatSpeed(int64(speed))
		}
		if eta, ok := stats["eta"].(float64); ok {
			progress.ETA = formatDuration(time.Duration(eta) * time.Second)
		}

		// Calculate percentage
		if progress.BytesTotal > 0 {
			progress.Percentage = float64(progress.BytesTransferred) / float64(progress.BytesTotal) * 100
		}

		task.Progress = progress
	}
}

// parseRclonePlainText parses plain text rclone output
func (te *TaskExecutor) parseRclonePlainText(task *TaskInfo, line string) {
	// Parse stats line like: "Transferred: 1.234 GiB / 5.678 GiB, 22%, 1.234 MiB/s, ETA 1h2m3s"
	if strings.Contains(line, "Transferred:") && strings.Contains(line, "%") {
		// This is a simplified parser - could be enhanced
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			// Extract percentage
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.HasSuffix(part, "%") {
					if task.Progress == nil {
						task.Progress = &TransferProgress{}
					}
					fmt.Sscanf(part, "%f%%", &task.Progress.Percentage)
					task.Progress.UpdatedAt = time.Now()
				}
			}
		}
	}
}

// CancelTask cancels a running task
func (te *TaskExecutor) CancelTask(executionID string) error {
	if val, ok := te.activeTasks.Load(executionID); ok {
		task := val.(*TaskInfo)
		if task.Cancel != nil {
			task.Cancel()
			task.Status = "cancelled"
			log.Printf("[Executor] Task %s cancelled", executionID)
			return nil
		}
	}
	return fmt.Errorf("task %s not found or not running", executionID)
}

// GetTaskStatus returns the status of a task
func (te *TaskExecutor) GetTaskStatus(executionID string) (*TaskInfo, bool) {
	if val, ok := te.activeTasks.Load(executionID); ok {
		return val.(*TaskInfo), true
	}
	return nil, false
}

// GetActiveTasks returns all active tasks
func (te *TaskExecutor) GetActiveTasks() []*TaskInfo {
	var tasks []*TaskInfo
	te.activeTasks.Range(func(key, value interface{}) bool {
		tasks = append(tasks, value.(*TaskInfo))
		return true
	})
	return tasks
}

// cleanupTaskWorkDir removes the task work directory
func (te *TaskExecutor) cleanupTaskWorkDir(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		log.Printf("[Executor] Warning: failed to cleanup work directory %s: %v", dir, err)
	}
}

// GetRclonePath returns the path to the rclone executable
func (te *TaskExecutor) GetRclonePath() string {
	return te.rcloneManager.GetExecutablePath()
}

// formatSpeed formats bytes per second as human readable
func formatSpeed(bps int64) string {
	const unit = 1024
	if bps < unit {
		return fmt.Sprintf("%d B/s", bps)
	}
	div, exp := int64(unit), 0
	for n := bps / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB/s", float64(bps)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration as human readable
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm%ds", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
}

func expandRcloneArgs(args []string) []string {
	var expanded []string
	for _, raw := range args {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		if !strings.ContainsAny(raw, " \t\r\n") {
			expanded = append(expanded, raw)
			continue
		}

		fields, err := splitCommandLine(raw)
		if err != nil || len(fields) == 0 {
			expanded = append(expanded, raw)
			continue
		}
		expanded = append(expanded, fields...)
	}
	return expanded
}

func splitCommandLine(input string) ([]string, error) {
	var (
		out      []string
		buf      strings.Builder
		inSingle bool
		inDouble bool
		escaped  bool
	)

	flush := func() {
		if buf.Len() == 0 {
			return
		}
		out = append(out, buf.String())
		buf.Reset()
	}

	for _, r := range input {
		if escaped {
			buf.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' && !inSingle {
			escaped = true
			continue
		}

		if r == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if r == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if !inSingle && !inDouble {
			if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
				flush()
				continue
			}
		}

		buf.WriteRune(r)
	}

	if escaped || inSingle || inDouble {
		return nil, fmt.Errorf("unterminated quote/escape in argument")
	}

	flush()
	return out, nil
}

func summarizeConfigForLog(configPath string) string {
	const remoteAlias = "remote"

	file, err := os.Open(configPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	type remoteSummary struct {
		Type          string
		Provider      string
		NoCheckBucket string
	}

	var summary remoteSummary
	currentSection := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimRight(scanner.Text(), "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if currentSection == remoteAlias && name != remoteAlias {
				break
			}
			currentSection = name
			continue
		}

		if currentSection != remoteAlias {
			continue
		}

		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(line[:eq]))
		value := strings.TrimSpace(line[eq+1:])

		switch key {
		case "type":
			summary.Type = value
		case "provider":
			summary.Provider = value
		case "no_check_bucket":
			summary.NoCheckBucket = value
		}
	}

	parts := make([]string, 0, 3)
	if summary.Type != "" {
		parts = append(parts, "type="+summary.Type)
	}
	if summary.Provider != "" {
		parts = append(parts, "provider="+summary.Provider)
	}
	if summary.NoCheckBucket != "" {
		parts = append(parts, "no_check_bucket="+summary.NoCheckBucket)
	}
	if len(parts) == 0 {
		return ""
	}

	return "[config] " + remoteAlias + "." + strings.Join(parts, " ")
}

func formatCommandForLog(exe string, args []string) string {
	safe := sanitizeArgsForLog(append([]string(nil), args...))

	parts := make([]string, 0, len(safe)+1)
	parts = append(parts, formatArgForLog(exe))
	for _, arg := range safe {
		parts = append(parts, formatArgForLog(arg))
	}
	return "[exec] " + strings.Join(parts, " ")
}

func sanitizeArgsForLog(args []string) []string {
	sensitiveFlags := map[string]struct{}{
		"--password":             {},
		"--password-command":     {},
		"--token":                {},
		"--access-key-id":        {},
		"--secret-access-key":    {},
		"--s3-access-key-id":     {},
		"--s3-secret-access-key": {},
		"--s3-session-token":     {},
	}

	isSensitiveFlag := func(flag string) bool {
		flag = strings.ToLower(strings.TrimSpace(flag))
		if _, ok := sensitiveFlags[flag]; ok {
			return true
		}
		if strings.Contains(flag, "secret") || strings.Contains(flag, "password") || strings.Contains(flag, "token") {
			return true
		}
		return false
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			continue
		}

		if eq := strings.Index(arg, "="); eq > 0 {
			flag := arg[:eq]
			if isSensitiveFlag(flag) {
				args[i] = flag + "=***"
			}
			continue
		}

		if isSensitiveFlag(arg) && i+1 < len(args) {
			args[i+1] = "***"
		}
	}

	return args
}

func formatArgForLog(arg string) string {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "''"
	}
	if strings.ContainsAny(arg, " \t\r\n\"'\\") {
		return strconv.Quote(arg)
	}
	return arg
}

func buildArchiveName(task *TaskInfo, archiveFormat string, now time.Time) string {
	baseName := sanitizeFilenameComponent(task.TaskName)
	if baseName == "" {
		baseName = sanitizeFilenameComponent(task.TaskID)
	}
	if baseName == "" {
		baseName = "backup"
	}

	archiveFormat = strings.TrimSpace(strings.TrimPrefix(archiveFormat, "."))
	if archiveFormat == "" {
		archiveFormat = "tar.gz"
	}

	return fmt.Sprintf("%s-%s.%s", baseName, now.Format("20060102150405"), archiveFormat)
}

func sanitizeFilenameComponent(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	input = strings.Join(strings.Fields(input), "-")

	const invalid = "/\\:*?\"<>|"
	const maxRunes = 80

	var buf strings.Builder
	buf.Grow(len(input))

	lastDash := false
	for _, r := range input {
		if r <= 31 || r == 127 {
			continue
		}

		if strings.ContainsRune(invalid, r) {
			if !lastDash {
				buf.WriteByte('-')
				lastDash = true
			}
			continue
		}

		if r == '-' {
			if lastDash {
				continue
			}
			lastDash = true
			buf.WriteRune(r)
			continue
		}

		lastDash = false
		buf.WriteRune(r)
	}

	out := strings.Trim(buf.String(), " .-_")
	if out == "" {
		return ""
	}

	runes := []rune(out)
	if len(runes) > maxRunes {
		out = string(runes[:maxRunes])
	}
	return out
}

type rcloneLSJSONEntry struct {
	Path  string `json:"Path"`
	Name  string `json:"Name"`
	IsDir bool   `json:"IsDir"`
}

type archiveCandidate struct {
	Name      string
	Timestamp time.Time
}

func (te *TaskExecutor) enforceArchiveRetention(ctx context.Context, task *TaskInfo, configPath, workDir string) error {
	maxRetention := task.MaxRetention
	if maxRetention <= 0 {
		return nil
	}

	destDir := strings.TrimSpace(task.DestPath)
	if destDir == "" {
		return nil
	}

	rclonePath := te.rcloneManager.GetExecutablePath()
	extraArgs := retentionRcloneArgs(task)

	lsArgs := append([]string{"lsjson", destDir, "--config", configPath, "--files-only", "--max-depth", "1"}, extraArgs...)
	if te.logHook != nil {
		te.logHook(task.ExecutionID, formatCommandForLog(rclonePath, lsArgs))
	}

	stdout, stderr, err := te.runRcloneCapture(ctx, workDir, configPath, lsArgs)
	if err != nil {
		msg := strings.TrimSpace(string(stderr))
		if msg == "" {
			msg = strings.TrimSpace(string(stdout))
		}
		if msg != "" {
			return fmt.Errorf("list remote backups failed: %s", msg)
		}
		return fmt.Errorf("list remote backups failed: %w", err)
	}

	var entries []rcloneLSJSONEntry
	if err := json.Unmarshal(stdout, &entries); err != nil {
		return fmt.Errorf("parse remote listing failed: %w", err)
	}

	candidates := make([]archiveCandidate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			name = strings.TrimSpace(entry.Path)
		}
		if name == "" {
			continue
		}

		ts, ok := parseArchiveTimestamp(name)
		if !ok {
			continue
		}
		candidates = append(candidates, archiveCandidate{Name: name, Timestamp: ts})
	}

	if len(candidates) <= maxRetention {
		if te.logHook != nil {
			te.logHook(task.ExecutionID, fmt.Sprintf("[retention] keep=%d found=%d delete=0", maxRetention, len(candidates)))
		}
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Timestamp.Equal(candidates[j].Timestamp) {
			return candidates[i].Name > candidates[j].Name
		}
		return candidates[i].Timestamp.After(candidates[j].Timestamp)
	})

	toDelete := candidates[maxRetention:]
	if te.logHook != nil {
		te.logHook(task.ExecutionID, fmt.Sprintf("[retention] keep=%d found=%d delete=%d", maxRetention, len(candidates), len(toDelete)))
	}

	var deleteErr error
	for _, candidate := range toDelete {
		target := joinRclonePath(destDir, candidate.Name)
		delArgs := append([]string{"deletefile", target, "--config", configPath}, extraArgs...)
		if te.logHook != nil {
			te.logHook(task.ExecutionID, formatCommandForLog(rclonePath, delArgs))
			te.logHook(task.ExecutionID, fmt.Sprintf("[retention] deleting %s", target))
		}

		_, stderr, err := te.runRcloneCapture(ctx, workDir, configPath, delArgs)
		if err != nil {
			msg := strings.TrimSpace(string(stderr))
			if msg == "" {
				msg = err.Error()
			}
			if te.logHook != nil {
				te.logHook(task.ExecutionID, fmt.Sprintf("[retention] delete failed: %s", msg))
			}
			deleteErr = fmt.Errorf("delete old backups failed: %w", err)
		}
	}

	return deleteErr
}

func retentionRcloneArgs(task *TaskInfo) []string {
	expanded := expandRcloneArgs(task.RcloneArgs)
	out := make([]string, 0, 1)
	for _, arg := range expanded {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			continue
		}
		if arg == "--s3-no-check-bucket" || strings.HasPrefix(arg, "--s3-no-check-bucket=") {
			out = append(out, arg)
		}
	}
	return out
}

func parseArchiveTimestamp(filename string) (time.Time, bool) {
	lower := strings.ToLower(strings.TrimSpace(filename))
	if lower == "" {
		return time.Time{}, false
	}

	suffixes := []string{".tar.gz", ".zip", ".tgz"}
	var suffix string
	for _, s := range suffixes {
		if strings.HasSuffix(lower, s) {
			suffix = s
			break
		}
	}
	if suffix == "" {
		return time.Time{}, false
	}

	base := filename[:len(filename)-len(suffix)]
	dash := strings.LastIndex(base, "-")
	if dash < 0 {
		return time.Time{}, false
	}

	ts := strings.TrimSpace(base[dash+1:])
	if len(ts) != 14 {
		return time.Time{}, false
	}

	parsed, err := time.ParseInLocation("20060102150405", ts, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func joinRclonePath(dir, name string) string {
	dir = strings.TrimRight(strings.TrimSpace(dir), "/")
	name = strings.TrimLeft(strings.TrimSpace(name), "/")
	if dir == "" {
		return name
	}
	if name == "" {
		return dir
	}
	return dir + "/" + name
}

func (te *TaskExecutor) runRcloneCapture(ctx context.Context, workDir, configPath string, args []string) ([]byte, []byte, error) {
	rclonePath := te.rcloneManager.GetExecutablePath()
	cmd := exec.CommandContext(ctx, rclonePath, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("RCLONE_CONFIG=%s", configPath),
		"RCLONE_NO_CHECK_CERTIFICATE=false",
		fmt.Sprintf("TMPDIR=%s", workDir),
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), stderr.Bytes(), err
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}
