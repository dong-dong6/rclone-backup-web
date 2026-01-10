package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

func (te *TaskExecutor) prepareDatabaseDump(ctx context.Context, task *TaskInfo, taskWorkDir string) (string, error) {
	engine := ""
	if task.DBEngine != nil {
		engine = strings.ToLower(strings.TrimSpace(*task.DBEngine))
	}
	if engine == "" {
		return "", fmt.Errorf("database task missing db_engine")
	}

	switch engine {
	case "postgres":
		if dbDumpMode(task) == "all" {
			return te.preparePostgresDumpAll(ctx, task, taskWorkDir)
		}
		return te.preparePostgresDump(ctx, task, taskWorkDir)
	case "mysql":
		if dbDumpMode(task) == "all" {
			return te.prepareMySQLDumpAll(ctx, task, taskWorkDir)
		}
		return te.prepareMySQLDump(ctx, task, taskWorkDir)
	case "sqlite":
		return te.prepareSQLiteDump(ctx, task, taskWorkDir)
	default:
		return "", fmt.Errorf("unsupported db_engine: %s", engine)
	}
}

func dbDumpMode(task *TaskInfo) string {
	if task == nil || task.DBDumpMode == nil {
		return "single"
	}
	mode := strings.ToLower(strings.TrimSpace(*task.DBDumpMode))
	if mode == "" {
		return "single"
	}
	return mode
}

func (te *TaskExecutor) preparePostgresDump(ctx context.Context, task *TaskInfo, taskWorkDir string) (string, error) {
	host, user, name, port, err := normalizeDBConn(task, "postgres")
	if err != nil {
		return "", err
	}

	exe, err := exec.LookPath("pg_dump")
	if err != nil {
		return "", fmt.Errorf("pg_dump not found: please install PostgreSQL client tools on the agent host")
	}

	outPath := filepath.Join(taskWorkDir, "db.sql")
	outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	args := []string{
		"--host", host,
		"--port", port,
		"--username", user,
		"--dbname", name,
		"--no-owner",
		"--no-privileges",
	}

	env := []string{}
	if strings.TrimSpace(task.DBPassword) != "" {
		env = append(env, "PGPASSWORD="+task.DBPassword)
	}

	if te.logHook != nil {
		te.logHook(task.ExecutionID, fmt.Sprintf("[db] pg_dump %s@%s:%s/%s -> %s", user, host, port, name, outPath))
		te.logHook(task.ExecutionID, formatCommandForLog(exe, args))
	}

	if err := te.runExternalCommandToFile(ctx, task.ExecutionID, taskWorkDir, env, exe, args, outFile); err != nil {
		return "", fmt.Errorf("pg_dump failed: %w", err)
	}

	if te.logHook != nil {
		te.logHook(task.ExecutionID, fmt.Sprintf("[db] pg_dump completed: %s", outPath))
	}
	return outPath, nil
}

func (te *TaskExecutor) preparePostgresDumpAll(ctx context.Context, task *TaskInfo, taskWorkDir string) (string, error) {
	host, user, port, err := normalizeDBConnNoName(task, "postgres")
	if err != nil {
		return "", err
	}

	exe, err := exec.LookPath("pg_dumpall")
	if err != nil {
		return "", fmt.Errorf("pg_dumpall not found: please install PostgreSQL client tools on the agent host")
	}

	outPath := filepath.Join(taskWorkDir, "db.sql")
	outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	args := []string{
		"--host", host,
		"--port", port,
		"--username", user,
		"--no-role-passwords",
	}

	env := []string{}
	if strings.TrimSpace(task.DBPassword) != "" {
		env = append(env, "PGPASSWORD="+task.DBPassword)
	}
	if task.DBName != nil && strings.TrimSpace(*task.DBName) != "" {
		env = append(env, "PGDATABASE="+strings.TrimSpace(*task.DBName))
	}

	if te.logHook != nil {
		te.logHook(task.ExecutionID, fmt.Sprintf("[db] pg_dumpall %s@%s:%s -> %s", user, host, port, outPath))
		te.logHook(task.ExecutionID, formatCommandForLog(exe, args))
		te.logHook(task.ExecutionID, "[db] note: pg_dumpall uses --no-role-passwords (role password hashes are not exported)")
	}

	if err := te.runExternalCommandToFile(ctx, task.ExecutionID, taskWorkDir, env, exe, args, outFile); err != nil {
		return "", fmt.Errorf("pg_dumpall failed: %w", err)
	}

	if te.logHook != nil {
		te.logHook(task.ExecutionID, fmt.Sprintf("[db] pg_dumpall completed: %s", outPath))
	}
	return outPath, nil
}

func (te *TaskExecutor) prepareMySQLDump(ctx context.Context, task *TaskInfo, taskWorkDir string) (string, error) {
	host, user, name, port, err := normalizeDBConn(task, "mysql")
	if err != nil {
		return "", err
	}

	exe, err := exec.LookPath("mysqldump")
	if err != nil {
		if alt, errAlt := exec.LookPath("mariadb-dump"); errAlt == nil {
			exe = alt
		} else {
			return "", fmt.Errorf("mysqldump not found: please install MySQL/MariaDB client tools on the agent host")
		}
	}

	outPath := filepath.Join(taskWorkDir, "db.sql")
	outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	args := []string{
		"--host", host,
		"--port", port,
		"--user", user,
		"--databases", name,
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
	}

	env := []string{}
	if strings.TrimSpace(task.DBPassword) != "" {
		// Avoid leaking the password via argv (still visible to child process).
		env = append(env, "MYSQL_PWD="+task.DBPassword)
	}

	if te.logHook != nil {
		te.logHook(task.ExecutionID, fmt.Sprintf("[db] mysqldump %s@%s:%s/%s -> %s", user, host, port, name, outPath))
		te.logHook(task.ExecutionID, formatCommandForLog(exe, args))
	}

	if err := te.runExternalCommandToFile(ctx, task.ExecutionID, taskWorkDir, env, exe, args, outFile); err != nil {
		return "", fmt.Errorf("mysqldump failed: %w", err)
	}

	if te.logHook != nil {
		te.logHook(task.ExecutionID, fmt.Sprintf("[db] mysqldump completed: %s", outPath))
	}
	return outPath, nil
}

func (te *TaskExecutor) prepareMySQLDumpAll(ctx context.Context, task *TaskInfo, taskWorkDir string) (string, error) {
	host, user, port, err := normalizeDBConnNoName(task, "mysql")
	if err != nil {
		return "", err
	}

	exe, err := exec.LookPath("mysqldump")
	if err != nil {
		if alt, errAlt := exec.LookPath("mariadb-dump"); errAlt == nil {
			exe = alt
		} else {
			return "", fmt.Errorf("mysqldump not found: please install MySQL/MariaDB client tools on the agent host")
		}
	}

	outPath := filepath.Join(taskWorkDir, "db.sql")
	outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	args := []string{
		"--host", host,
		"--port", port,
		"--user", user,
		"--all-databases",
		"--single-transaction",
		"--quick",
		"--lock-tables=false",
	}

	env := []string{}
	if strings.TrimSpace(task.DBPassword) != "" {
		env = append(env, "MYSQL_PWD="+task.DBPassword)
	}

	if te.logHook != nil {
		te.logHook(task.ExecutionID, fmt.Sprintf("[db] mysqldump --all-databases %s@%s:%s -> %s", user, host, port, outPath))
		te.logHook(task.ExecutionID, formatCommandForLog(exe, args))
	}

	if err := te.runExternalCommandToFile(ctx, task.ExecutionID, taskWorkDir, env, exe, args, outFile); err != nil {
		return "", fmt.Errorf("mysqldump failed: %w", err)
	}

	if te.logHook != nil {
		te.logHook(task.ExecutionID, fmt.Sprintf("[db] mysqldump completed: %s", outPath))
	}
	return outPath, nil
}

func (te *TaskExecutor) prepareSQLiteDump(ctx context.Context, task *TaskInfo, taskWorkDir string) (string, error) {
	if task.DBPath == nil || strings.TrimSpace(*task.DBPath) == "" {
		return "", fmt.Errorf("sqlite task missing db_path")
	}
	sourcePath := filepath.Clean(strings.TrimSpace(*task.DBPath))

	outPath := filepath.Join(taskWorkDir, "db.sqlite3")

	if exe, err := exec.LookPath("sqlite3"); err == nil {
		args := []string{"-cmd", ".timeout 5000", sourcePath, ".backup " + outPath}
		if te.logHook != nil {
			te.logHook(task.ExecutionID, fmt.Sprintf("[db] sqlite3 backup %s -> %s", sourcePath, outPath))
			te.logHook(task.ExecutionID, formatCommandForLog(exe, args))
		}

		if err := te.runExternalCommand(ctx, task.ExecutionID, taskWorkDir, nil, exe, args); err != nil {
			return "", fmt.Errorf("sqlite3 backup failed: %w", err)
		}

		_ = os.Chmod(outPath, 0600)
		if te.logHook != nil {
			te.logHook(task.ExecutionID, fmt.Sprintf("[db] sqlite3 backup completed: %s", outPath))
		}
		return outPath, nil
	}

	if te.logHook != nil {
		te.logHook(task.ExecutionID, "[db] warning: sqlite3 not found, falling back to file copy (may be inconsistent if db is busy)")
	}

	src, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if err := copyWithContext(ctx, dst, src); err != nil {
		return "", err
	}

	if te.logHook != nil {
		te.logHook(task.ExecutionID, fmt.Sprintf("[db] sqlite file copied: %s", outPath))
	}
	return outPath, nil
}

func normalizeDBConn(task *TaskInfo, engine string) (host, user, name, port string, err error) {
	if task.DBHost == nil || strings.TrimSpace(*task.DBHost) == "" {
		return "", "", "", "", fmt.Errorf("%s task missing db_host", engine)
	}
	if task.DBUser == nil || strings.TrimSpace(*task.DBUser) == "" {
		return "", "", "", "", fmt.Errorf("%s task missing db_user", engine)
	}
	if task.DBName == nil || strings.TrimSpace(*task.DBName) == "" {
		return "", "", "", "", fmt.Errorf("%s task missing db_name", engine)
	}

	host = strings.TrimSpace(*task.DBHost)
	user = strings.TrimSpace(*task.DBUser)
	name = strings.TrimSpace(*task.DBName)

	p := 0
	if task.DBPort != nil {
		p = *task.DBPort
	}
	if p <= 0 {
		if engine == "postgres" {
			p = 5432
		} else {
			p = 3306
		}
	}
	port = fmt.Sprintf("%d", p)
	return host, user, name, port, nil
}

func normalizeDBConnNoName(task *TaskInfo, engine string) (host, user, port string, err error) {
	if task.DBHost == nil || strings.TrimSpace(*task.DBHost) == "" {
		return "", "", "", fmt.Errorf("%s task missing db_host", engine)
	}
	if task.DBUser == nil || strings.TrimSpace(*task.DBUser) == "" {
		return "", "", "", fmt.Errorf("%s task missing db_user", engine)
	}

	host = strings.TrimSpace(*task.DBHost)
	user = strings.TrimSpace(*task.DBUser)

	p := 0
	if task.DBPort != nil {
		p = *task.DBPort
	}
	if p <= 0 {
		if engine == "postgres" {
			p = 5432
		} else {
			p = 3306
		}
	}
	port = fmt.Sprintf("%d", p)
	return host, user, port, nil
}

func (te *TaskExecutor) runExternalCommand(ctx context.Context, executionID, workDir string, env []string, exe string, args []string) error {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		te.streamCommandOutput(executionID, stdout, "")
	}()
	go func() {
		defer wg.Done()
		te.streamCommandOutput(executionID, stderr, "[stderr] ")
	}()
	waitErr := cmd.Wait()
	wg.Wait()
	return waitErr
}

func (te *TaskExecutor) runExternalCommandToFile(ctx context.Context, executionID, workDir string, env []string, exe string, args []string, stdout io.Writer) error {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = stdout

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		te.streamCommandOutput(executionID, stderr, "[stderr] ")
	}()
	waitErr := cmd.Wait()
	wg.Wait()
	return waitErr
}

func (te *TaskExecutor) streamCommandOutput(executionID string, reader io.Reader, prefix string) {
	if reader == nil {
		return
	}

	if te.logHook == nil {
		_, _ = io.Copy(io.Discard, reader)
		return
	}

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	scanner.Split(scanLinesCRLF)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		te.logHook(executionID, prefix+line)
	}
	_, _ = io.Copy(io.Discard, reader)
}
