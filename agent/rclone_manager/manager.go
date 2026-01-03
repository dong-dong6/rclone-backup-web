package rclone_manager

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	RcloneVersion         = "v1.67.0"
	RcloneDownloadRetries = 3
	RcloneDownloadTimeout = 5 * time.Minute
)

type Manager struct {
	workDir        string
	executablePath string
	version        string
}

// NewManager creates a new rclone manager
func NewManager(workDir string) *Manager {
	return &Manager{
		workDir: workDir,
		version: RcloneVersion,
	}
}

// EnsureRcloneExists checks if rclone exists and downloads it if not
func (m *Manager) EnsureRcloneExists() (string, error) {
	// Determine the executable name based on OS
	execName := "rclone"
	if runtime.GOOS == "windows" {
		execName = "rclone.exe"
	}

	m.executablePath = filepath.Join(m.workDir, "bin", execName)

	// Check if already exists
	if m.isRcloneValid() {
		log.Printf("[RcloneManager] Found existing rclone executable: %s", m.executablePath)
		return m.executablePath, nil
	}

	log.Printf("[RcloneManager] Rclone not found or invalid. Starting download...")

	// Create bin directory
	binDir := filepath.Dir(m.executablePath)
	if err := os.MkdirAll(binDir, 0755); err != nil {
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
		return m.executablePath, nil
	}

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
	// Determine download URL
	url, err := m.getRcloneDownloadURL()
	if err != nil {
		return fmt.Errorf("failed to determine download URL: %w", err)
	}

	log.Printf("[RcloneManager] Downloading from: %s", url)

	// Create temporary file for download
	tmpFile, err := os.CreateTemp("", "rclone-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download with timeout
	client := &http.Client{
		Timeout: RcloneDownloadTimeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Copy to temp file with progress reporting
	written, err := m.copyWithProgress(tmpFile, resp.Body, resp.ContentLength)
	if err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}

	log.Printf("[RcloneManager] Downloaded %d bytes", written)

	expectedChecksum, allowInsecure := m.getExpectedChecksum()
	if expectedChecksum != "" {
		if err := m.VerifyChecksum(tmpFile.Name(), expectedChecksum); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		log.Printf("[RcloneManager] Checksum verified for %s", url)
	} else if !allowInsecure {
		return fmt.Errorf("missing expected checksum for rclone download; set RCLONE_CHECKSUM or RCLONE_CHECKSUM_%s_%s, or set RCLONE_ALLOW_INSECURE_DOWNLOADS=true to bypass (not recommended)", strings.ToUpper(runtime.GOOS), strings.ToUpper(runtime.GOARCH))
	} else {
		log.Printf("[RcloneManager] Skipping checksum verification (RCLONE_ALLOW_INSECURE_DOWNLOADS=true)")
	}

	// Extract rclone executable from zip
	if err := m.extractRclone(tmpFile.Name()); err != nil {
		return fmt.Errorf("failed to extract: %w", err)
	}

	return nil
}

// getRcloneDownloadURL constructs the download URL for the current platform
func (m *Manager) getRcloneDownloadURL() (string, error) {
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
		return "", fmt.Errorf("unsupported architecture: %s", goarch)
	}

	// Construct download URL
	// Format: https://github.com/rclone/rclone/releases/download/v1.67.0/rclone-v1.67.0-linux-amd64.zip
	zipName := fmt.Sprintf("rclone-%s-%s-%s.zip", m.version, goos, rcloneArch)
	url := fmt.Sprintf("https://github.com/rclone/rclone/releases/download/%s/%s", m.version, zipName)

	return url, nil
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

// Cleanup removes the rclone executable
func (m *Manager) Cleanup() error {
	if m.executablePath == "" {
		return nil
	}

	log.Printf("[RcloneManager] Cleaning up rclone at: %s", m.executablePath)
	return os.Remove(m.executablePath)
}
