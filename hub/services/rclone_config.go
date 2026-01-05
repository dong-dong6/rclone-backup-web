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
// - A config body without any [section] header (common UI format)
// - A full config section like:
//   [myremote]
//   type = s3
//   ...
//
// If multiple sections are present, it prefers the [remote] section; otherwise it uses the
// first section and re-homes its key/value lines under [remote].
func NormalizeRcloneConfigForSingleRemote(config string) string {
	config = strings.TrimSpace(config)
	if config == "" {
		return ""
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
		lines := sections[DefaultRcloneRemoteAlias]
		if lines == nil {
			lines = sections[firstSection]
		}
		content = strings.TrimSpace(strings.Join(lines, "\n"))
	}

	if content == "" {
		return "[" + DefaultRcloneRemoteAlias + "]"
	}
	return "[" + DefaultRcloneRemoteAlias + "]\n" + content
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

