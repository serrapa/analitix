package providerutil

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/serrapa/analitix/pkg/targets"
)

func ParseDomainsFromBody(body []byte) []string {
	bodyText := strings.TrimSpace(string(body))
	if bodyText == "" {
		return nil
	}

	if domains := parseDomainsFromJSON(body); len(domains) > 0 {
		return NormalizeDomains(domains)
	}

	lines := strings.Split(bodyText, "\n")
	var domains []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if idx := strings.Index(line, ","); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if strings.HasPrefix(strings.ToLower(line), "error") {
			continue
		}
		domains = append(domains, line)
	}
	return NormalizeDomains(domains)
}

func NormalizeDomains(inputs []string) []string {
	unique := make(map[string]struct{})
	var domains []string
	for _, input := range inputs {
		asset := targets.AssetFromInput(input)
		if asset == "" {
			continue
		}
		if _, ok := unique[asset]; ok {
			continue
		}
		unique[asset] = struct{}{}
		domains = append(domains, asset)
	}
	sort.Strings(domains)
	return domains
}

func parseDomainsFromJSON(body []byte) []string {
	type domainRow struct {
		Domain string `json:"domain"`
		Host   string `json:"host"`
		Value  string `json:"value"`
	}

	var rows []domainRow
	if err := json.Unmarshal(body, &rows); err == nil {
		var items []string
		for _, row := range rows {
			switch {
			case row.Domain != "":
				items = append(items, row.Domain)
			case row.Host != "":
				items = append(items, row.Host)
			case row.Value != "":
				items = append(items, row.Value)
			}
		}
		return items
	}

	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil
	}

	for _, key := range []string{"results", "domains", "data", "items"} {
		raw, ok := wrapped[key]
		if !ok {
			continue
		}

		var values []string
		if err := json.Unmarshal(raw, &values); err == nil {
			return values
		}

		var nested []domainRow
		if err := json.Unmarshal(raw, &nested); err == nil {
			var items []string
			for _, row := range nested {
				switch {
				case row.Domain != "":
					items = append(items, row.Domain)
				case row.Host != "":
					items = append(items, row.Host)
				case row.Value != "":
					items = append(items, row.Value)
				}
			}
			return items
		}
	}

	return nil
}
