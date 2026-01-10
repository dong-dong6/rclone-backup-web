package rclone_manager

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	RcloneVersion         = "v1.67.0"
	RcloneDownloadRetries = 3
	RcloneDownloadTimeout = 60 * time.Minute
)

type Manager struct {
	workDir        string
	executablePath string
	version        string

	ensureMu       sync.Mutex
	ensuring       bool
	ensureWait     chan struct{}
	lastEnsureErr  error
}

// NewManager creates a new rclone manager
func NewManager(workDir string) *Manager {
	execName := "rclone"
	if runtime.GOOS == "windows" {
		execName = "rclone.exe"
	}

	return &Manager{
		workDir:        workDir,
		executablePath: filepath.Join(workDir, "bin", execName),
		version:        RcloneVersion,
	}
}

// EnsureRcloneExists checks if rclone exists and downloads it if not
func (m *Manager) EnsureRcloneExists() (string, error) {
	if m.isRcloneValid() {
		log.Printf("[RcloneManager] Found existing rclone executable: %s", m.executablePath)
		return m.executablePath, nil
	}

	m.ensureMu.Lock()
	if m.isRcloneValid() {
		m.ensureMu.Unlock()
		log.Printf("[RcloneManager] Found existing rclone executable: %s", m.executablePath)
		return m.executablePath, nil
	}

	if m.ensuring {
		wait := m.ensureWait
		m.ensureMu.Unlock()
		if wait != nil {
			<-wait
		}
		if m.isRcloneValid() {
			log.Printf("[RcloneManager] Found existing rclone executable: %s", m.executablePath)
			return m.executablePath, nil
		}
		m.ensureMu.Lock()
		err := m.lastEnsureErr
		m.ensureMu.Unlock()
		if err == nil {
			err = fmt.Errorf("rclone is still missing after ensure")
		}
		return "", err
	}

	m.ensuring = true
	wait := make(chan struct{})
	m.ensureWait = wait
	m.ensureMu.Unlock()

	log.Printf("[RcloneManager] Rclone not found or invalid. Starting download...")

	// Create bin directory
	binDir := filepath.Dir(m.executablePath)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		m.ensureMu.Lock()
		m.lastEnsureErr = err
		m.ensuring = false
		close(wait)
		m.ensureMu.Unlock()
		return "", fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Download with retries
	var lastErr error
	for i := 0; i < RcloneDownloadRetries; i++ {
		if i > 0 {
			log.Printf("[RcloneManager] Retry %d/%d...", i+1, RcloneDownloadRetries)
			time.Sleep(time.Second * time.Duration(i*2)) // Exponential backoff
		}

		if err := m.downloadRclone(); err != nil {
			lastErr = err
			log.Printf("[RcloneManager] Download attempt failed: %v", err)
			continue
		}

		log.Printf("[RcloneManager] Rclone downloaded and installed successfully at: %s", m.executablePath)
		m.ensureMu.Lock()
		m.lastEnsureErr = nil
		m.ensuring = false
		close(wait)
		m.ensureMu.Unlock()
		return m.executablePath, nil
	}

	m.ensureMu.Lock()
	m.lastEnsureErr = lastErr
	m.ensuring = false
	close(wait)
	m.ensureMu.Unlock()

	return "", fmt.Errorf("failed after %d retries: %w", RcloneDownloadRetries, lastErr)
}

// GetExecutablePath returns the path to the rclone executable
func (m *Manager) GetExecutablePath() string {
	return m.executablePath
}

// isRcloneValid checks if the rclone executable exists and is valid
func (m *Manager) isRcloneValid() bool {
	info, err := os.Stat(m.executablePath)
	if err != nil {
		return false
	}

	// Check if it's a regular file and executable
	if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
		// TODO: Could add version check here by running "rclone version"
		return true
	}

	return false
}

// downloadRclone downloads and extracts rclone
func (m *Manager) downloadRclone() error {
	downloadURLs, zipName, err := m.getRcloneDownloadURLs()
	if err != nil {
		return fmt.Errorf("failed to determine download URL: %w", err)
	}

	var lastErr error
	for _, downloadURL := range downloadURLs {
		downloadURL = strings.TrimSpace(downloadURL)
		if downloadURL == "" {
			continue
		}

		log.Printf("[RcloneManager] Downloading from: %s", downloadURL)

		// Create temporary file for download
		tmpFile, err := os.CreateTemp("", "rclone-*.zip")
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		tmpPath := tmpFile.Name()

		// Download with timeout
		client := &http.Client{
			Timeout: RcloneDownloadTimeout,
		}

		resp, err := client.Get(downloadURL)
		if err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
			lastErr = fmt.Errorf("failed to download: %w", err)
			log.Printf("[RcloneManager] Download attempt failed: %v", lastErr)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
			lastErr = fmt.Errorf("download failed with status: %s", resp.Status)
			log.Printf("[RcloneManager] Download attempt failed: %v", lastErr)
			continue
		}

		// Copy to temp file with progress reporting
		written, err := m.copyWithProgress(tmpFile, resp.Body, resp.ContentLength)
		_ = resp.Body.Close()
		if err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
			lastErr = fmt.Errorf("failed to save download: %w", err)
			log.Printf("[RcloneManager] Download attempt failed: %v", lastErr)
			continue
		}

		log.Printf("[RcloneManager] Downloaded %d bytes", written)

		if err := tmpFile.Close(); err != nil {
			_ = os.Remove(tmpPath)
			lastErr = fmt.Errorf("failed to finalize download: %w", err)
			log.Printf("[RcloneManager] Download attempt failed: %v", lastErr)
			continue
		}

		requireChecksum := strings.EqualFold(strings.TrimSpace(os.Getenv("RCLONE_REQUIRE_CHECKSUM")), "true")
		expectedChecksum, allowInsecure := m.getExpectedChecksum()
		if allowInsecure && !requireChecksum && expectedChecksum == "" {
			log.Printf("[RcloneManager] Skipping checksum verification (RCLONE_ALLOW_INSECURE_DOWNLOADS=true)")
		} else {
			checksum := expectedChecksum
			if checksum == "" {
				if fetched, err := m.fetchChecksum(downloadURL, zipName); err == nil {
					checksum = fetched
				} else if requireChecksum {
					_ = os.Remove(tmpPath)
					lastErr = fmt.Errorf("checksum required but not available: %w", err)
					log.Printf("[RcloneManager] Download attempt failed: %v", lastErr)
					continue
				} else {
					log.Printf("[RcloneManager] Warning: checksum unavailable, proceeding without verification (set RCLONE_REQUIRE_CHECKSUM=true to enforce)")
				}
			}

			if checksum != "" {
				if err := m.VerifyChecksum(tmpPath, checksum); err != nil {
					_ = os.Remove(tmpPath)
					lastErr = fmt.Errorf("checksum verification failed: %w", err)
					log.Printf("[RcloneManager] Download attempt failed: %v", lastErr)
					continue
				}
				log.Printf("[RcloneManager] Checksum verified for %s", downloadURL)
			}
		}

		// Extract rclone executable from zip
		if err := m.extractRclone(tmpPath); err != nil {
			_ = os.Remove(tmpPath)
			lastErr = fmt.Errorf("failed to extract: %w", err)
			log.Printf("[RcloneManager] Download attempt failed: %v", lastErr)
			continue
		}

		_ = os.Remove(tmpPath)
		lastErr = nil

		if lastErr == nil {
			return nil
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no download URLs available")
	}
	return lastErr
}

func (m *Manager) getRcloneDownloadURLs() ([]string, string, error) {
	if value := strings.TrimSpace(os.Getenv("RCLONE_DOWNLOAD_URLS")); value != "" {
		parts := strings.Split(value, ",")
		urls := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				urls = append(urls, part)
			}
		}
		if len(urls) > 0 {
			return urls, "", nil
		}
	}

	if value := strings.TrimSpace(os.Getenv("RCLONE_DOWNLOAD_URL")); value != "" {
		return []string{value}, "", nil
	}

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map Go arch to rclone release arch
	archMap := map[string]string{
		"amd64": "amd64",
		"386":   "386",
		"arm64": "arm64",
		"arm":   "arm-v7",
	}

	rcloneArch, ok := archMap[goarch]
	if !ok {
		return nil, "", fmt.Errorf("unsupported architecture: %s", goarch)
	}

	// Construct download URL
	// Format: https://github.com/rclone/rclone/releases/download/v1.67.0/rclone-v1.67.0-linux-amd64.zip
	zipName := fmt.Sprintf("rclone-%s-%s-%s.zip", m.version, goos, rcloneArch)
	downloadURLs := []string{
		fmt.Sprintf("https://downloads.rclone.org/%s", zipName),
	}

	githubURL := fmt.Sprintf("https://github.com/rclone/rclone/releases/download/%s/%s", m.version, zipName)
	if proxy := strings.TrimSpace(os.Getenv("RCLONE_GITHUB_PROXY")); proxy != "" {
		proxy = strings.TrimRight(proxy, "/")
		downloadURLs = append(downloadURLs, proxy+"/"+githubURL)
	}
	downloadURLs = append(downloadURLs, githubURL)

	return downloadURLs, zipName, nil
}

// copyWithProgress copies from src to dst with progress reporting
func (m *Manager) copyWithProgress(dst io.Writer, src io.Reader, total int64) (int64, error) {
	buf := make([]byte, 32*1024) // 32KB buffer
	var written int64
	var lastReport time.Time

	for {
		nr, err := src.Read(buf)
		if nr > 0 {
			nw, err := dst.Write(buf[0:nr])
			if err != nil {
				return written, err
			}
			written += int64(nw)

			// Report progress every second
			if time.Since(lastReport) > time.Second && total > 0 {
				percentage := float64(written) / float64(total) * 100
				log.Printf("[RcloneManager] Download progress: %.1f%%", percentage)
				lastReport = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return written, err
		}
	}

	return written, nil
}

// extractRclone extracts the rclone executable from the zip file
func (m *Manager) extractRclone(zipPath string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer reader.Close()

	execName := "rclone"
	if runtime.GOOS == "windows" {
		execName = "rclone.exe"
	}

	// Find and extract rclone executable
	for _, file := range reader.File {
		// The executable is typically at: rclone-v1.67.0-linux-amd64/rclone
		if strings.HasSuffix(file.Name, execName) && !strings.Contains(file.Name, ".1") {
			log.Printf("[RcloneManager] Found rclone in zip: %s", file.Name)

			// Open the file in zip
			rc, err := file.Open()
			if err != nil {
				return fmt.Errorf("failed to open file in zip: %w", err)
			}
			defer rc.Close()

			// Create the output file
			outFile, err := os.OpenFile(m.executablePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("failed to create output file: %w", err)
			}
			defer outFile.Close()

			// Copy the executable
			if _, err := io.Copy(outFile, rc); err != nil {
				return fmt.Errorf("failed to extract executable: %w", err)
			}

			// Ensure executable permissions on Unix
			if runtime.GOOS != "windows" {
				if err := os.Chmod(m.executablePath, 0755); err != nil {
					return fmt.Errorf("failed to set executable permissions: %w", err)
				}
			}

			return nil
		}
	}

	return fmt.Errorf("rclone executable not found in zip")
}

// VerifyChecksum verifies the SHA256 checksum of the downloaded file
func (m *Manager) VerifyChecksum(filePath, expectedChecksum string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

func (m *Manager) getExpectedChecksum() (string, bool) {
	envKey := fmt.Sprintf("RCLONE_CHECKSUM_%s_%s", strings.ToUpper(runtime.GOOS), strings.ToUpper(runtime.GOARCH))

	if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
		return strings.ToLower(value), false
	}

	if value := strings.TrimSpace(os.Getenv("RCLONE_CHECKSUM")); value != "" {
		return strings.ToLower(value), false
	}

	allowInsecure := strings.EqualFold(os.Getenv("RCLONE_ALLOW_INSECURE_DOWNLOADS"), "true")
	return "", allowInsecure
}

func (m *Manager) fetchChecksum(downloadURL, zipName string) (string, error) {
	downloadURL = strings.TrimSpace(downloadURL)
	if downloadURL == "" {
		return "", fmt.Errorf("download url is empty")
	}

	zipName = strings.TrimSpace(zipName)
	if zipName == "" {
		if parsed, parseErr := url.Parse(downloadURL); parseErr == nil && strings.TrimSpace(parsed.Path) != "" {
			zipName = path.Base(parsed.Path)
		} else {
			zipName = path.Base(downloadURL)
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	shaURL := downloadURL + ".sha256"
	if checksum, err := fetchChecksumFromSHAFile(client, shaURL); err == nil && checksum != "" {
		return checksum, nil
	}

	u, err := url.Parse(downloadURL)
	if err != nil {
		return "", fmt.Errorf("invalid download url: %w", err)
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = path.Join(path.Dir(u.Path), "SHA256SUMS")

	checksum, err := fetchChecksumFromSHA256SUMS(client, u.String(), zipName)
	if err != nil {
		return "", err
	}
	if checksum == "" {
		return "", fmt.Errorf("checksum not found")
	}
	return checksum, nil
}

func fetchChecksumFromSHAFile(client *http.Client, shaURL string) (string, error) {
	resp, err := client.Get(shaURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("sha256 url returned %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return "", err
	}

	line := strings.TrimSpace(string(body))
	if line == "" {
		return "", fmt.Errorf("sha256 response empty")
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", fmt.Errorf("sha256 response malformed")
	}

	sum := strings.TrimSpace(fields[0])
	sum = strings.TrimPrefix(sum, "SHA256:")
	sum = strings.TrimPrefix(sum, "sha256:")
	sum = strings.TrimSpace(sum)
	if len(sum) != 64 {
		return "", fmt.Errorf("sha256 checksum length invalid")
	}
	return strings.ToLower(sum), nil
}

func fetchChecksumFromSHA256SUMS(client *http.Client, sumsURL string, zipName string) (string, error) {
	resp, err := client.Get(sumsURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SHA256SUMS url returned %s", resp.Status)
	}

	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 8*1024*1024))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		sum := strings.TrimSpace(fields[0])
		name := strings.TrimSpace(fields[len(fields)-1])
		name = strings.TrimPrefix(name, "*")
		if name != zipName {
			continue
		}

		if len(sum) != 64 {
			continue
		}
		return strings.ToLower(sum), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("checksum not found for %s", zipName)
}

// Cleanup removes the rclone executable
func (m *Manager) Cleanup() error {
	if m.executablePath == "" {
		return nil
	}

	log.Printf("[RcloneManager] Cleaning up rclone at: %s", m.executablePath)
	return os.Remove(m.executablePath)
}
