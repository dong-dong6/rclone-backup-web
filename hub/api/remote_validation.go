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

func validateRemotePresetConfig(presetKey string, remoteName string, configData string) error {
	parsed := parseSingleRemoteConfig(configData)
	if parsed.SectionCount != 1 {
		return fmt.Errorf("config_data must contain exactly one [section]")
	}
	if parsed.SectionName != remoteName {
		return fmt.Errorf("config_data section name must match remote name")
	}

	typ := strings.TrimSpace(parsed.Options["type"])
	if typ == "" {
		return fmt.Errorf("config_data is missing type")
	}

	requireOption := func(key string) error {
		if strings.TrimSpace(parsed.Options[key]) == "" {
			return fmt.Errorf("config_data is missing %s", key)
		}
		return nil
	}

	switch strings.TrimSpace(presetKey) {
	case "drive":
		if typ != "drive" {
			return fmt.Errorf("preset_key drive requires type = drive")
		}
		if err := requireOption("token"); err != nil {
			return err
		}
		return nil

	case "onedrive":
		if typ != "onedrive" {
			return fmt.Errorf("preset_key onedrive requires type = onedrive")
		}
		if err := requireOption("token"); err != nil {
			return err
		}
		return nil

	case "s3_cloudflare_r2":
		if typ != "s3" {
			return fmt.Errorf("preset_key s3_cloudflare_r2 requires type = s3")
		}
		if strings.TrimSpace(parsed.Options["provider"]) != "Cloudflare" {
			return fmt.Errorf("preset_key s3_cloudflare_r2 requires provider = Cloudflare")
		}
		if err := requireOption("access_key_id"); err != nil {
			return err
		}
		if err := requireOption("secret_access_key"); err != nil {
			return err
		}
		if err := requireOption("endpoint"); err != nil {
			return err
		}
		return nil

	case "s3_alibaba_oss":
		if typ != "s3" {
			return fmt.Errorf("preset_key s3_alibaba_oss requires type = s3")
		}
		if strings.TrimSpace(parsed.Options["provider"]) != "Alibaba" {
			return fmt.Errorf("preset_key s3_alibaba_oss requires provider = Alibaba")
		}
		if err := requireOption("access_key_id"); err != nil {
			return err
		}
		if err := requireOption("secret_access_key"); err != nil {
			return err
		}
		if err := requireOption("endpoint"); err != nil {
			return err
		}
		return nil

	case "s3_tencent_cos":
		if typ != "s3" {
			return fmt.Errorf("preset_key s3_tencent_cos requires type = s3")
		}
		if strings.TrimSpace(parsed.Options["provider"]) != "TencentCOS" {
			return fmt.Errorf("preset_key s3_tencent_cos requires provider = TencentCOS")
		}
		if err := requireOption("access_key_id"); err != nil {
			return err
		}
		if err := requireOption("secret_access_key"); err != nil {
			return err
		}
		if err := requireOption("endpoint"); err != nil {
			return err
		}
		return nil

	default:
		return fmt.Errorf("unknown preset_key: %s", strings.TrimSpace(presetKey))
	}
}
