package providerutil

import "strings"

func PreviewBody(body []byte, max int) string {
	if max <= 0 {
		max = 200
	}
	text := strings.TrimSpace(string(body))
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > max {
		return text[:max] + "..."
	}
	return text
}
