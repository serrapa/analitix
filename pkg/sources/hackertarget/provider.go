package hackertarget

import (
	"context"
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

const endpoint = "https://api.hackertarget.com/analyticslookup/"

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "hackertarget"
}

func (p *Provider) IsDefault() bool {
	return true
}

func (p *Provider) NeedsKey() bool {
	return false
}

func (p *Provider) DefaultRateLimit() *types.RateLimitSpec {
	return &types.RateLimitSpec{MaxCount: 1, Duration: time.Second}
}

func (p *Provider) Stages() []types.StageType {
	return []types.StageType{types.StageReverse}
}

func (p *Provider) SupportsStage(stage types.StageType) bool {
	return stage == types.StageReverse
}

func (p *Provider) Execute(ctx context.Context, runSession *session.Client, request types.ProviderRequest) ([]types.Observation, error) {
	values := url.Values{}
	values.Set("q", request.Value)

	body, response, err := runSession.Get(ctx, p.Name(), endpoint+"?"+values.Encode())
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusMultipleChoices {
		gologger.Verbose().Msgf("provider=%s status=%d query=%s body_preview=%q", p.Name(), response.StatusCode, request.Value, providerutil.PreviewBody(body, 220))
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("hackertarget rate limited the request with status %d", response.StatusCode)
	}
	if response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("hackertarget returned status %d", response.StatusCode)
	}

	bodyText := strings.TrimSpace(string(body))
	lowerBody := strings.ToLower(bodyText)
	if strings.Contains(lowerBody, "api count exceeded") || strings.Contains(lowerBody, "daily limit") {
		return nil, fmt.Errorf("hackertarget quota exceeded")
	}
	if strings.Contains(lowerBody, "no records found") || bodyText == "" {
		gologger.Verbose().Msgf("provider=%s query=%s no_results body_preview=%q", p.Name(), request.Value, providerutil.PreviewBody(body, 220))
		return nil, nil
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
			Evidence:     fmt.Sprintf("%s?q=%s", endpoint, url.QueryEscape(request.Value)),
		})
	}

	return observations, nil
}
