package api

import (
	"fmt"
	"strings"
)

func validateRemoteName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("remote name is required")
	}

	if strings.ContainsAny(trimmed, "\r\n") {
		return "", fmt.Errorf("remote name must be a single line")
	}

	// Prevent breaking the ini section header and rclone remote syntax (<name>:).
	if strings.ContainsAny(trimmed, "[]:") {
		return "", fmt.Errorf("remote name contains invalid characters")
	}

	if len(trimmed) > 128 {
		return "", fmt.Errorf("remote name is too long")
	}

	return trimmed, nil
}

type parsedRemoteConfig struct {
	SectionName  string
	SectionCount int
	Options      map[string]string
}

func parseSingleRemoteConfig(configData string) parsedRemoteConfig {
	lines := strings.Split(configData, "\n")

	var sectionName string
	sectionCount := 0
	options := make(map[string]string)

	inSection := false

	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.TrimRight(rawLine, "\r"))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") && len(line) >= 2 {
			sectionCount++
			sectionName = strings.TrimSpace(line[1 : len(line)-1])
			inSection = true
			continue
		}

		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}

		key := strings.TrimSpace(line[:eq])
		value := strings.TrimSpace(line[eq+1:])
		if key == "" {
			continue
		}

		if inSection && sectionCount == 1 {
			options[key] = value
		}
	}

	return parsedRemoteConfig{
		SectionName:  sectionName,
		SectionCount: sectionCount,
		Options:      options,
	}
}

func validateRemoteConfig(remoteName string, configData string) (remoteType string, err error) {
	parsed := parseSingleRemoteConfig(configData)
	if parsed.SectionCount != 1 {
		return "", fmt.Errorf("config_data must contain exactly one [section]")
	}
	if parsed.SectionName != remoteName {
		return "", fmt.Errorf("config_data section name must match remote name")
	}

	typ := strings.TrimSpace(parsed.Options["type"])
	if typ == "" {
		return "", fmt.Errorf("config_data is missing type")
	}

	return typ, nil
}
