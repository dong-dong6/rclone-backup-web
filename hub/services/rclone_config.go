package services

import (
	"bufio"
	"strings"
)

const DefaultRcloneRemoteAlias = "remote"

// NormalizeRcloneConfigForSingleRemote returns a config file content that defines exactly one
// rclone remote named "remote" (DefaultRcloneRemoteAlias).
//
// Supported inputs:
//   - A config body without any [section] header (common UI format)
//   - A full config section like:
//     [myremote]
//     type = s3
//     ...
//
// If multiple sections are present, it prefers the [remote] section; otherwise it uses the
// first section and re-homes its key/value lines under [remote].
func NormalizeRcloneConfigForSingleRemote(config string) string {
	return NormalizeRcloneConfigForRemoteAlias(config, DefaultRcloneRemoteAlias)
}

// NormalizeRcloneConfigForRemoteAlias returns a config file content that defines exactly one
// rclone remote named alias.
//
// It follows the same parsing rules as NormalizeRcloneConfigForSingleRemote, but lets callers
// pick the section name (useful when the agent should execute with the user's remote name).
func NormalizeRcloneConfigForRemoteAlias(config string, alias string) string {
	config = strings.TrimSpace(config)
	if config == "" {
		return ""
	}

	alias = strings.TrimSpace(alias)
	if alias == "" {
		alias = DefaultRcloneRemoteAlias
	}

	var (
		hasSections  bool
		firstSection string
		current      string
		bodyLines    []string
		sections     = map[string][]string{}
	)

	scanner := bufio.NewScanner(strings.NewReader(config))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if name, ok := parseRcloneSectionHeader(trimmed); ok {
			hasSections = true
			current = name
			if firstSection == "" {
				firstSection = name
			}
			continue
		}

		if !hasSections {
			bodyLines = append(bodyLines, line)
			continue
		}

		// Ignore any content before the first header (should be rare).
		if current == "" {
			continue
		}
		sections[current] = append(sections[current], line)
	}

	var content string
	if !hasSections {
		content = strings.TrimSpace(strings.Join(bodyLines, "\n"))
	} else {
		lines := sections[alias]
		if lines == nil {
			lines = sections[firstSection]
		}
		content = strings.TrimSpace(strings.Join(lines, "\n"))
	}

	if content == "" {
		return "[" + alias + "]"
	}
	content = maybeEnableS3NoCheckBucket(content)
	return "[" + alias + "]\n" + content
}

func maybeEnableS3NoCheckBucket(content string) string {
	options := parseRcloneOptions(content)
	if !strings.EqualFold(strings.TrimSpace(options["type"]), "s3") {
		return content
	}
	if !strings.EqualFold(strings.TrimSpace(options["provider"]), "Cloudflare") {
		return content
	}
	if _, ok := options["no_check_bucket"]; ok {
		return content
	}
	return content + "\nno_check_bucket = true"
}

func parseRcloneOptions(content string) map[string]string {
	options := make(map[string]string)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimRight(scanner.Text(), "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(line[:eq]))
		value := strings.TrimSpace(line[eq+1:])
		if key == "" {
			continue
		}
		options[key] = value
	}

	return options
}

func parseRcloneSectionHeader(trimmedLine string) (string, bool) {
	if !strings.HasPrefix(trimmedLine, "[") {
		return "", false
	}

	end := strings.IndexByte(trimmedLine, ']')
	if end <= 1 {
		return "", false
	}

	name := strings.TrimSpace(trimmedLine[1:end])
	if name == "" {
		return "", false
	}

	rest := strings.TrimSpace(trimmedLine[end+1:])
	if rest == "" || strings.HasPrefix(rest, "#") || strings.HasPrefix(rest, ";") {
		return name, true
	}

	return "", false
}
