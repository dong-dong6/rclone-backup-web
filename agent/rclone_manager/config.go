package rclone_manager

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigManager manages rclone configurations
type ConfigManager struct {
	workDir string
}

// NewConfigManager creates a new config manager
func NewConfigManager(workDir string) *ConfigManager {
	return &ConfigManager{
		workDir: workDir,
	}
}

// CreateTempConfig creates a temporary rclone config file for a task
func (cm *ConfigManager) CreateTempConfig(taskID string, configContent string) (string, error) {
	// Create task-specific config directory
	configDir := filepath.Join(cm.workDir, "configs", taskID)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	
	configPath := filepath.Join(configDir, "rclone.conf")
	
	// Decode config if it's base64 encoded
	var configData []byte
	if decoded, err := base64.StdEncoding.DecodeString(configContent); err == nil {
		configData = decoded
	} else {
		configData = []byte(configContent)
	}
	
	// Write config file with restricted permissions
	if err := os.WriteFile(configPath, configData, 0600); err != nil {
		return "", fmt.Errorf("failed to write config file: %w", err)
	}
	
	return configPath, nil
}

// CleanupConfig removes temporary config for a task
func (cm *ConfigManager) CleanupConfig(taskID string) error {
	configDir := filepath.Join(cm.workDir, "configs", taskID)
	return os.RemoveAll(configDir)
}

// ValidateConfig checks if a config file is valid
func (cm *ConfigManager) ValidateConfig(configPath string) error {
	info, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("config file not found: %w", err)
	}
	
	// Check permissions (should be readable only by owner)
	if info.Mode().Perm()&0077 != 0 {
		return fmt.Errorf("config file has insecure permissions: %v", info.Mode().Perm())
	}
	
	return nil
}