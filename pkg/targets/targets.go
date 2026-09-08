package targets

import (
	"net/url"
	"regexp"
	"strings"
)

var hostnamePattern = regexp.MustCompile(`^[a-z0-9.-]+$`)

func AssetFromInput(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err == nil && parsed.Hostname() != "" {
			return normalizeHostname(parsed.Hostname())
		}
	}

	raw = strings.TrimPrefix(raw, "https://")
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "/")
	if idx := strings.Index(raw, "/"); idx >= 0 {
		raw = raw[:idx]
	}
	if idx := strings.Index(raw, ":"); idx >= 0 {
		raw = raw[:idx]
	}
	return normalizeHostname(raw)
}

func LiveCandidates(raw string) ([]string, string) {
	asset := AssetFromInput(raw)
	if asset == "" {
		return nil, ""
	}

	if strings.Contains(raw, "://") {
		return []string{strings.TrimSpace(raw)}, asset
	}

	return []string{
		"https://" + asset,
		"http://" + asset,
	}, asset
}

func normalizeHostname(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Trim(value, ".")
	if value == "" || strings.Contains(value, " ") {
		return ""
	}
	if !hostnamePattern.MatchString(value) {
		return ""
	}
	if !strings.Contains(value, ".") {
		return ""
	}
	return value
}
