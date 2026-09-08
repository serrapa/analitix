package runner

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/serrapa/analitix/pkg/targets"
	"github.com/serrapa/analitix/pkg/types"
)

type collector struct {
	plan    *types.ExecutionPlan
	errors  []types.ProviderError
	items   map[string]*types.Finding
	stats   map[string]*types.ProviderStat
	started time.Time
	mu      sync.Mutex
}

func newCollector(plan *types.ExecutionPlan) *collector {
	return &collector{
		plan:    plan,
		items:   make(map[string]*types.Finding),
		stats:   make(map[string]*types.ProviderStat),
		started: time.Now(),
	}
}

func (c *collector) AddError(err types.ProviderError) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = append(c.errors, err)
}

func (c *collector) AddObservations(observations ...types.Observation) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, observation := range observations {
		key := strings.ToLower(observation.Asset) + "|" + strings.ToUpper(observation.TrackingID) + "|" + string(observation.TrackingType)
		item, ok := c.items[key]
		if !ok {
			item = &types.Finding{
				Asset:        observation.Asset,
				AssetType:    observation.AssetType,
				TrackingID:   observation.TrackingID,
				TrackingType: observation.TrackingType,
			}
			c.items[key] = item
		}

		item.Current = item.Current || observation.Current
		item.Historical = item.Historical || observation.Historical
		item.FirstSeen = minTime(item.FirstSeen, observation.FirstSeen)
		item.LastSeen = maxTime(item.LastSeen, observation.LastSeen)

		item.Inputs = appendUniqueString(item.Inputs, observation.Input)
		item.Queries = appendUniqueString(item.Queries, observation.Query)
		item.Sources = appendUniqueString(item.Sources, observation.Source)
		item.Stages = appendUniqueStage(item.Stages, observation.Stage)
		item.Evidence = appendUniqueString(item.Evidence, observation.Evidence)
	}
}

func (c *collector) AddStat(provider string, stage types.StageType, requests int, observations int, duration time.Duration, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item := c.ensureStat(provider, stage)
	item.Requests += requests
	item.Observations += observations
	item.Duration += duration
	if item.Requests > 0 {
		item.AverageLatency = item.Duration / time.Duration(item.Requests)
	}
	if err != nil {
		item.Errors++
	}
}

func (c *collector) MarkSkipped(provider string, stage types.StageType, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item := c.ensureStat(provider, stage)
	item.Skipped = true
	item.SkipReason = reason
}

func (c *collector) ReverseRequests(params types.ProviderParams) []types.ProviderRequest {
	findings := c.Result().Findings
	requests := make([]types.ProviderRequest, 0, len(findings))

	for _, finding := range findings {
		requests = append(requests, types.ProviderRequest{
			Query:        strings.Join(finding.Queries, ","),
			Value:        finding.TrackingID,
			TrackingID:   finding.TrackingID,
			TrackingType: finding.TrackingType,
			Params:       params,
		})
	}

	return requests
}

func (c *collector) Result() *types.ExecutionResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	findings := make([]types.Finding, 0, len(c.items))
	for _, item := range c.items {
		copyItem := *item
		sort.Strings(copyItem.Inputs)
		sort.Strings(copyItem.Queries)
		sort.Strings(copyItem.Sources)
		sort.Strings(copyItem.Evidence)
		sort.Slice(copyItem.Stages, func(i, j int) bool {
			return copyItem.Stages[i] < copyItem.Stages[j]
		})
		findings = append(findings, copyItem)
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Asset == findings[j].Asset {
			return findings[i].TrackingID < findings[j].TrackingID
		}
		return findings[i].Asset < findings[j].Asset
	})

	errs := append([]types.ProviderError(nil), c.errors...)
	sort.Slice(errs, func(i, j int) bool {
		if errs[i].Provider == errs[j].Provider {
			return errs[i].Input < errs[j].Input
		}
		return errs[i].Provider < errs[j].Provider
	})

	stats := make([]types.ProviderStat, 0, len(c.stats))
	for _, stat := range c.stats {
		copyStat := *stat
		stats = append(stats, copyStat)
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Provider == stats[j].Provider {
			return stats[i].Stage < stats[j].Stage
		}
		return stats[i].Provider < stats[j].Provider
	})

	for findingIndex := range findings {
		for _, source := range findings[findingIndex].Sources {
			for statIndex := range stats {
				if stats[statIndex].Provider == source {
					stats[statIndex].Findings++
				}
			}
		}
	}

	finished := time.Now()
	return &types.ExecutionResult{
		Plan:     c.plan,
		Findings: findings,
		Errors:   errs,
		Stats: types.ExecutionStats{
			StartedAt:  c.started,
			FinishedAt: finished,
			Duration:   finished.Sub(c.started),
			Providers:  stats,
		},
	}
}

func (c *collector) ensureStat(provider string, stage types.StageType) *types.ProviderStat {
	key := provider + "|" + string(stage)
	item, ok := c.stats[key]
	if !ok {
		item = &types.ProviderStat{Provider: provider, Stage: stage}
		c.stats[key] = item
	}
	return item
}

func buildProviderParams(options *Options) (types.ProviderParams, error) {
	startDate, err := parseDateTime(options.StartDate)
	if err != nil {
		return types.ProviderParams{}, fmt.Errorf("invalid start date: %w", err)
	}
	endDate, err := parseDateTime(options.EndDate)
	if err != nil {
		return types.ProviderParams{}, fmt.Errorf("invalid end date: %w", err)
	}

	return types.ProviderParams{
		StartDate: startDate,
		EndDate:   endDate,
		Frequency: strings.TrimSpace(strings.ToLower(options.Frequency)),
		Limit:     options.Limit,
	}, nil
}

func domainProviderRequests(inputs []string, params types.ProviderParams) []types.ProviderRequest {
	requests := make([]types.ProviderRequest, 0, len(inputs))
	for _, input := range inputs {
		asset := targets.AssetFromInput(input)
		requests = append(requests, types.ProviderRequest{
			Query:  asset,
			Value:  input,
			Params: params,
		})
	}
	return requests
}

func directTrackingRequests(inputs []string, params types.ProviderParams) []types.ProviderRequest {
	requests := make([]types.ProviderRequest, 0, len(inputs))
	for _, input := range inputs {
		requests = append(requests, types.ProviderRequest{
			Query:        strings.TrimSpace(input),
			Value:        strings.TrimSpace(input),
			TrackingID:   strings.TrimSpace(input),
			TrackingType: types.ClassifyTrackingID(input),
			Params:       params,
		})
	}
	return requests
}

func parseDateTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04",
		"02/01/2006",
		"02/01/2006:15:04",
	}
	for _, layout := range layouts {
		value, err := time.Parse(layout, raw)
		if err == nil {
			return &value, nil
		}
	}

	return nil, fmt.Errorf("unsupported date format %q", raw)
}

func minTime(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.Before(*current) {
		value := *candidate
		return &value
	}
	return current
}

func maxTime(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.After(*current) {
		value := *candidate
		return &value
	}
	return current
}

func appendUniqueString(items []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return items
	}
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func appendUniqueStage(items []types.StageType, value types.StageType) []types.StageType {
	if value == "" {
		return items
	}
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}
