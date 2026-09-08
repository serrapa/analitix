package dnslytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/projectdiscovery/gologger"
	"github.com/serrapa/analitix/pkg/providerutil"
	"github.com/serrapa/analitix/pkg/session"
	"github.com/serrapa/analitix/pkg/types"
)

const endpoint = "https://api.dnslytics.net/v1/reverseganalytics/"

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "dnslytics"
}

func (p *Provider) IsDefault() bool {
	return true
}

func (p *Provider) NeedsKey() bool {
	return true
}

func (p *Provider) DefaultRateLimit() *types.RateLimitSpec {
	return nil
}

func (p *Provider) Stages() []types.StageType {
	return []types.StageType{types.StageReverse}
}

func (p *Provider) SupportsStage(stage types.StageType) bool {
	return stage == types.StageReverse
}

func (p *Provider) Execute(ctx context.Context, runSession *session.Client, request types.ProviderRequest) ([]types.Observation, error) {
	apiKey := runSession.ProviderConfig.APIKey(p.Name())
	if apiKey == "" {
		return nil, fmt.Errorf("missing api_key")
	}

	values := url.Values{}
	values.Set("apikey", apiKey)

	body, response, err := runSession.Get(ctx, p.Name(), endpoint+url.PathEscape(request.Value)+"?"+values.Encode())
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusMultipleChoices {
		gologger.Verbose().Msgf("provider=%s status=%d query=%s body_preview=%q", p.Name(), response.StatusCode, request.Value, providerutil.PreviewBody(body, 220))
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("dnslytics rejected the api key with status %d", response.StatusCode)
	}
	if response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("dnslytics returned status %d", response.StatusCode)
	}

	records, err := parseResponse(body)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		gologger.Verbose().Msgf("provider=%s query=%s no_results body_preview=%q", p.Name(), request.Value, providerutil.PreviewBody(body, 220))
		return nil, nil
	}

	observations := make([]types.Observation, 0, len(records))
	for _, record := range records {
		observations = append(observations, types.Observation{
			Input:        request.Value,
			Query:        request.Query,
			Asset:        record.Domain,
			AssetType:    types.AssetTypeDomain,
			TrackingID:   request.TrackingID,
			TrackingType: request.TrackingType,
			Source:       p.Name(),
			Stage:        types.StageReverse,
			Historical:   record.FirstSeen != nil || record.LastSeen != nil,
			FirstSeen:    record.FirstSeen,
			LastSeen:     record.LastSeen,
			Evidence:     endpoint,
		})
	}

	return observations, nil
}

type domainRecord struct {
	Domain    string
	FirstSeen *time.Time
	LastSeen  *time.Time
}

type responseEnvelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

type responseData struct {
	Domains []json.RawMessage `json:"domains"`
}

type domainObject struct {
	Domain    string `json:"domain"`
	FirstSeen string `json:"firstseen"`
	LastSeen  string `json:"lastseen"`
}

func parseResponse(body []byte) ([]domainRecord, error) {
	var envelope responseEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("could not decode dnslytics response: %w", err)
	}

	status := strings.ToLower(strings.TrimSpace(envelope.Status))
	if status != "success" && status != "ok" {
		var message string
		_ = json.Unmarshal(envelope.Data, &message)
		if message == "" {
			message = fmt.Sprintf("dnslytics returned status %q", envelope.Status)
		}
		return nil, fmt.Errorf("%s", message)
	}

	var data responseData
	if err := json.Unmarshal(envelope.Data, &data); err != nil {
		return nil, fmt.Errorf("could not decode dnslytics data: %w", err)
	}

	var records []domainRecord
	for _, raw := range data.Domains {
		var domain string
		if err := json.Unmarshal(raw, &domain); err == nil {
			records = append(records, domainRecord{Domain: domain})
			continue
		}

		var row domainObject
		if err := json.Unmarshal(raw, &row); err == nil {
			records = append(records, domainRecord{
				Domain:    row.Domain,
				FirstSeen: parseDate(row.FirstSeen),
				LastSeen:  parseDate(row.LastSeen),
			})
		}
	}

	normalized := make([]domainRecord, 0, len(records))
	for _, record := range records {
		domains := providerutil.NormalizeDomains([]string{record.Domain})
		if len(domains) == 0 {
			continue
		}
		record.Domain = domains[0]
		normalized = append(normalized, record)
	}

	return normalized, nil
}

func parseDate(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
		value, err := time.Parse(layout, raw)
		if err == nil {
			return &value
		}
	}
	return nil
}
