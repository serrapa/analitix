package osintsh

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/projectdiscovery/gologger"
	"github.com/serrapa/analitix/pkg/providerutil"
	"github.com/serrapa/analitix/pkg/session"
	"github.com/serrapa/analitix/pkg/types"
)

const endpoint = "https://api.osint.sh/ga"

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "osintsh"
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
	values.Set("q", request.Value)
	values.Set("api_key", apiKey)

	body, response, err := runSession.Get(ctx, p.Name(), endpoint+"?"+values.Encode())
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusMultipleChoices {
		gologger.Verbose().Msgf("provider=%s status=%d query=%s body_preview=%q", p.Name(), response.StatusCode, request.Value, providerutil.PreviewBody(body, 220))
	}
	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("osintsh rejected the api key with status %d", response.StatusCode)
	}
	if response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("osintsh returned status %d", response.StatusCode)
	}

	bodyText := strings.TrimSpace(string(body))
	lowerBody := strings.ToLower(bodyText)
	if bodyText == "" || strings.Contains(lowerBody, "no results") {
		gologger.Verbose().Msgf("provider=%s query=%s no_results body_preview=%q", p.Name(), request.Value, providerutil.PreviewBody(body, 220))
		return nil, nil
	}
	if strings.Contains(lowerBody, "invalid api key") || strings.Contains(lowerBody, "unauthorized") {
		return nil, fmt.Errorf("osintsh reported an authentication error")
	}

	domains := providerutil.ParseDomainsFromBody(body)
	if len(domains) == 0 {
		gologger.Verbose().Msgf("provider=%s query=%s no_domains_extracted body_preview=%q", p.Name(), request.Value, providerutil.PreviewBody(body, 220))
		return nil, nil
	}

	observations := make([]types.Observation, 0, len(domains))
	for _, domain := range domains {
		observations = append(observations, types.Observation{
			Input:        request.Value,
			Query:        request.Query,
			Asset:        domain,
			AssetType:    types.AssetTypeDomain,
			TrackingID:   request.TrackingID,
			TrackingType: request.TrackingType,
			Source:       p.Name(),
			Stage:        types.StageReverse,
			Evidence:     endpoint,
		})
	}

	return observations, nil
}
