package wayback

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/projectdiscovery/gologger"
	"github.com/serrapa/analitix/pkg/extract"
	"github.com/serrapa/analitix/pkg/providerutil"
	"github.com/serrapa/analitix/pkg/session"
	"github.com/serrapa/analitix/pkg/targets"
	"github.com/serrapa/analitix/pkg/types"
)

const cdxEndpoint = "https://web.archive.org/cdx/search/cdx"

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "wayback"
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
	return []types.StageType{types.StageHistory}
}

func (p *Provider) SupportsStage(stage types.StageType) bool {
	return stage == types.StageHistory
}

func (p *Provider) Execute(ctx context.Context, runSession *session.Client, request types.ProviderRequest) ([]types.Observation, error) {
	asset := targets.AssetFromInput(request.Value)
	if asset == "" {
		return nil, fmt.Errorf("could not derive a domain from %q", request.Value)
	}

	snapshots, err := p.fetchSnapshots(ctx, runSession, asset, request.Params)
	if err != nil {
		return nil, err
	}

	var observations []types.Observation
	for _, snapshot := range snapshots {
		body, response, err := runSession.Get(ctx, p.Name(), snapshot.ArchiveURL())
		if err != nil {
			continue
		}
		if response.StatusCode >= http.StatusMultipleChoices {
			gologger.Verbose().Msgf("provider=%s status=%d archive_url=%s body_preview=%q", p.Name(), response.StatusCode, snapshot.ArchiveURL(), providerutil.PreviewBody(body, 220))
			continue
		}

		detected := extract.ExtractParityTrackingIDs(body)
		if len(detected) == 0 {
			gologger.Verbose().Msgf("provider=%s archive_url=%s no_tracking_ids_extracted body_preview=%q", p.Name(), snapshot.ArchiveURL(), providerutil.PreviewBody(body, 220))
		}
		for _, item := range detected {
			timestamp := snapshot.Time()
			observations = append(observations, types.Observation{
				Input:        request.Value,
				Query:        request.Query,
				Asset:        asset,
				AssetType:    types.AssetTypeDomain,
				TrackingID:   item.Value,
				TrackingType: item.Type,
				Source:       p.Name(),
				Stage:        types.StageHistory,
				Historical:   true,
				FirstSeen:    timestamp,
				LastSeen:     timestamp,
				Evidence:     snapshot.ArchiveURL(),
			})
		}
	}

	return observations, nil
}

type snapshotRow struct {
	Timestamp string
	Original  string
}

func (s snapshotRow) ArchiveURL() string {
	return "https://web.archive.org/web/" + s.Timestamp + "id_/" + s.Original
}

func (s snapshotRow) Time() *time.Time {
	value, err := time.Parse("20060102150405", s.Timestamp)
	if err != nil {
		return nil
	}
	return &value
}

func (p *Provider) fetchSnapshots(ctx context.Context, runSession *session.Client, asset string, params types.ProviderParams) ([]snapshotRow, error) {
	values := url.Values{}
	values.Set("url", asset)
	values.Set("matchType", "domain")
	values.Set("filter", "statuscode:200")
	values.Set("fl", "timestamp,original")
	values.Set("output", "json")
	values.Set("limit", fmt.Sprintf("%d", params.Limit))

	if collapse := collapseValue(params.Frequency); collapse != "" {
		values.Set("collapse", collapse)
	}
	if params.StartDate != nil {
		values.Set("from", params.StartDate.Format("20060102150405"))
	}
	if params.EndDate != nil {
		values.Set("to", params.EndDate.Format("20060102150405"))
	}

	body, response, err := runSession.Get(ctx, p.Name(), cdxEndpoint+"?"+values.Encode())
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("wayback cdx returned status %d body=%q", response.StatusCode, providerutil.PreviewBody(body, 220))
	}

	var rows [][]string
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("could not decode cdx response: %w", err)
	}

	snapshots := make([]snapshotRow, 0, len(rows))
	for index, row := range rows {
		if index == 0 || len(row) < 2 {
			continue
		}
		snapshots = append(snapshots, snapshotRow{
			Timestamp: strings.TrimSpace(row[0]),
			Original:  strings.TrimSpace(row[1]),
		})
	}

	return snapshots, nil
}

func collapseValue(frequency string) string {
	switch strings.TrimSpace(strings.ToLower(frequency)) {
	case "yearly", "year":
		return "timestamp:4"
	case "monthly", "month":
		return "timestamp:6"
	case "daily", "day":
		return "timestamp:8"
	case "hourly", "hour":
		return "timestamp:10"
	default:
		return ""
	}
}
