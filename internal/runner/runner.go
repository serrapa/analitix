package runner

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/projectdiscovery/gologger"
	"github.com/serrapa/analitix/pkg/extract"
	"github.com/serrapa/analitix/pkg/output"
	"github.com/serrapa/analitix/pkg/providerconfig"
	"github.com/serrapa/analitix/pkg/session"
	"github.com/serrapa/analitix/pkg/sources"
	"github.com/serrapa/analitix/pkg/targets"
	"github.com/serrapa/analitix/pkg/types"
)

const internalLookupSource = "lookup"

type Runner struct {
	options  *Options
	registry *sources.Registry
}

func New(options *Options) (*Runner, error) {
	return &Runner{
		options:  options,
		registry: sources.NewRegistry(),
	}, nil
}

func (r *Runner) Run(ctx context.Context, writer io.Writer) error {
	if r.options.ListSources {
		return r.listSources(writer)
	}

	plan, err := r.BuildPlan()
	if err != nil {
		return err
	}

	if r.options.PlanOnly {
		return output.WritePlan(writer, plan, r.options.JSON || r.options.JSONL)
	}

	providerConfig, err := providerconfig.Load(r.options.ProviderConfig)
	if err != nil {
		return err
	}

	runSession := session.New(ctx, session.Config{
		Timeout:                   time.Duration(r.options.Timeout) * time.Second,
		RateLimit:                 r.options.RateLimit,
		RateLimits:                r.options.RateLimits.AsMap(),
		DefaultProviderRateLimits: r.registry.DefaultRateLimits(),
		ProviderConfig:            providerConfig,
		Proxy:                     r.options.Proxy,
		Delay:                     time.Duration(r.options.Delay) * time.Second,
		Retries:                   r.options.Retries,
	})
	defer runSession.Close()

	execution, err := r.Execute(ctx, plan, runSession)
	if err != nil {
		return err
	}

	return output.WriteExecutionResult(writer, execution, r.options.JSON, r.options.JSONL, r.options.Stats)
}

func (r *Runner) BuildPlan() (*types.ExecutionPlan, error) {
	workflow, err := r.workflow()
	if err != nil {
		return nil, err
	}

	stageProviders, err := r.selectProviders(workflow)
	if err != nil {
		return nil, err
	}

	plan := &types.ExecutionPlan{
		Mode:       r.options.Mode,
		Workflow:   workflow,
		PlanOnly:   r.options.PlanOnly,
		Inputs:     append([]string(nil), planInputs(r.options)...),
		StagePlans: make([]types.StagePlan, 0, len(stageProviders)),
	}

	for _, stage := range []types.StageType{types.StageLookup, types.StageHistory, types.StageReverse} {
		if !r.shouldIncludeStage(plan, stage) {
			continue
		}
		if stage == types.StageLookup && r.usesInternalLookup(plan) {
			plan.StagePlans = append(plan.StagePlans, types.StagePlan{
				Stage:     stage,
				Providers: []string{internalLookupSource},
			})
			continue
		}

		providers := stageProviders[stage]
		if len(providers) == 0 {
			continue
		}

		names := make([]string, 0, len(providers))
		for _, item := range providers {
			names = append(names, item.Name())
		}
		sort.Strings(names)

		plan.StagePlans = append(plan.StagePlans, types.StagePlan{
			Stage:     stage,
			Providers: names,
		})
	}

	return plan, nil
}

func (r *Runner) listSources(writer io.Writer) error {
	all := r.registry.All()
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name() < all[j].Name()
	})

	if _, err := fmt.Fprintf(writer, "%-16s %-24s %-8s %s\n", "name", "stages", "auth", "default_rl"); err != nil {
		return err
	}

	for _, item := range all {
		auth := "none"
		if item.NeedsKey() {
			auth = "api-key"
		}
		defaultLimit := "-"
		if spec := item.DefaultRateLimit(); spec != nil {
			defaultLimit = fmt.Sprintf("%d/%s", spec.MaxCount, spec.Duration)
		}
		if _, err := fmt.Fprintf(writer, "%-16s %-24s %-8s %s\n", item.Name(), joinStages(item.Stages()), auth, defaultLimit); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) workflow() (types.WorkflowType, error) {
	switch r.options.Mode {
	case ModeLookup:
		return types.WorkflowLookup, nil
	case ModeHistory:
		return types.WorkflowHistory, nil
	case ModeReverse:
		return types.WorkflowReverse, nil
	default:
		return "", fmt.Errorf("unsupported mode %q", r.options.Mode)
	}
}

func (r *Runner) selectProviders(workflow types.WorkflowType) (map[types.StageType][]sources.Provider, error) {
	selected, err := r.registry.Select(sources.Selection{
		Workflow:         workflow,
		Include:          []string(r.options.Sources),
		Exclude:          []string(r.options.ExcludeSources),
		DirectTrackingID: len(r.options.TrackingIDs) > 0,
	})
	if err != nil {
		return nil, err
	}

	grouped := make(map[types.StageType][]sources.Provider)
	for _, item := range selected {
		for _, stage := range item.Stages() {
			grouped[stage] = append(grouped[stage], item)
		}
	}

	return grouped, nil
}

func planInputs(options *Options) []string {
	if len(options.TrackingIDs) > 0 {
		return []string(options.TrackingIDs)
	}

	return []string(options.URLs)
}

func (r *Runner) Execute(ctx context.Context, plan *types.ExecutionPlan, runSession *session.Client) (*types.ExecutionResult, error) {
	params, err := buildProviderParams(r.options)
	if err != nil {
		return nil, err
	}

	stageProviders, err := r.selectProviders(plan.Workflow)
	if err != nil {
		return nil, err
	}

	collector := newCollector(plan)
	var reverseInputs []types.ProviderRequest

	switch plan.Workflow {
	case types.WorkflowLookup:
		domainRequests := domainProviderRequests(r.options.URLs, params)
		if err := r.executeLookupStage(ctx, runSession, domainRequests, collector); err != nil {
			return nil, err
		}
	case types.WorkflowHistory:
		domainRequests := domainProviderRequests(r.options.URLs, params)
		if err := r.executeStage(ctx, runSession, types.StageHistory, stageProviders[types.StageHistory], domainRequests, collector); err != nil {
			return nil, err
		}
	case types.WorkflowReverse:
		if len(r.options.TrackingIDs) > 0 {
			reverseInputs = directTrackingRequests(r.options.TrackingIDs, params)
		} else {
			domainRequests := domainProviderRequests(r.options.URLs, params)
			if err := r.executeLookupStage(ctx, runSession, domainRequests, collector); err != nil {
				return nil, err
			}
			if r.reverseUsesHistory() {
				if err := r.executeStage(ctx, runSession, types.StageHistory, stageProviders[types.StageHistory], domainRequests, collector); err != nil {
					return nil, err
				}
			}
			reverseInputs = collector.ReverseRequests(params)
			if len(reverseInputs) == 0 {
				if r.reverseUsesHistory() {
					r.info("no tracking IDs discovered from lookup/history; reverse stage has no input")
				} else {
					r.info("no tracking IDs discovered from lookup; reverse stage has no input")
				}
			}
		}

		if len(reverseInputs) > 0 {
			if err := r.executeStage(ctx, runSession, types.StageReverse, stageProviders[types.StageReverse], reverseInputs, collector); err != nil {
				return nil, err
			}
		}
	}

	return collector.Result(), nil
}

func (r *Runner) reverseUsesHistory() bool {
	return r.options.ReverseDepth == ReverseDepthLookupHistory
}

func (r *Runner) usesInternalLookup(plan *types.ExecutionPlan) bool {
	switch plan.Workflow {
	case types.WorkflowLookup:
		return len(r.options.URLs) > 0
	case types.WorkflowReverse:
		return len(r.options.TrackingIDs) == 0 && len(r.options.URLs) > 0
	default:
		return false
	}
}

func (r *Runner) shouldIncludeStage(plan *types.ExecutionPlan, stage types.StageType) bool {
	if plan.Workflow != types.WorkflowReverse {
		return true
	}

	if len(r.options.TrackingIDs) > 0 {
		return stage == types.StageReverse
	}

	switch stage {
	case types.StageLookup:
		return true
	case types.StageHistory:
		return r.reverseUsesHistory()
	case types.StageReverse:
		return true
	default:
		return false
	}
}

func (r *Runner) executeLookupStage(ctx context.Context, runSession *session.Client, requests []types.ProviderRequest, collector *collector) error {
	if len(requests) == 0 {
		return nil
	}

	started := time.Now()
	r.infof("stage %s: starting with internal lookup over %d input(s)", types.StageLookup, len(requests))

	for _, request := range requests {
		candidates, asset := targets.LiveCandidates(request.Value)
		if len(candidates) == 0 {
			err := fmt.Errorf("could not derive a valid URL candidate from %q", request.Value)
			collector.AddStat(internalLookupSource, types.StageLookup, 1, 0, 0, err)
			collector.AddError(types.ProviderError{
				Provider: internalLookupSource,
				Query:    request.Query,
				Input:    request.Value,
				Message:  err.Error(),
			})
			continue
		}

		var lastErr error
		var observations []types.Observation
		requestStarted := time.Now()
		for _, candidate := range candidates {
			body, response, err := runSession.Get(ctx, internalLookupSource, candidate)
			if err != nil {
				lastErr = err
				r.verbosef("stage %s: source=%s candidate=%s request_error=%s", types.StageLookup, internalLookupSource, candidate, err)
				continue
			}
			if response.StatusCode >= http.StatusMultipleChoices {
				lastErr = fmt.Errorf("http status %d", response.StatusCode)
				r.verbosef("stage %s: source=%s candidate=%s unusual_status=%d body_preview=%q", types.StageLookup, internalLookupSource, candidate, response.StatusCode, previewBody(body, 220))
				continue
			}

			lastErr = nil
			detected := extract.ExtractParityTrackingIDs(body)
			observations = make([]types.Observation, 0, len(detected))
			for _, item := range detected {
				observations = append(observations, types.Observation{
					Input:        request.Value,
					Query:        request.Query,
					Asset:        asset,
					AssetType:    types.AssetTypeDomain,
					TrackingID:   item.Value,
					TrackingType: item.Type,
					Source:       internalLookupSource,
					Stage:        types.StageLookup,
					Current:      true,
					Evidence:     candidate,
				})
			}
			if len(observations) == 0 {
				r.verbosef("stage %s: source=%s candidate=%s no_tracking_ids_extracted body_preview=%q", types.StageLookup, internalLookupSource, candidate, previewBody(body, 220))
			}
			break
		}

		collector.AddStat(internalLookupSource, types.StageLookup, 1, len(observations), time.Since(requestStarted), lastErr)
		if lastErr != nil && len(observations) == 0 {
			collector.AddError(types.ProviderError{
				Provider: internalLookupSource,
				Query:    request.Query,
				Input:    request.Value,
				Message:  lastErr.Error(),
			})
			continue
		}
		collector.AddObservations(observations...)
	}

	r.infof("stage %s: finished in %s", types.StageLookup, time.Since(started).Round(time.Millisecond))
	return nil
}

func (r *Runner) executeStage(ctx context.Context, runSession *session.Client, stage types.StageType, providers []sources.Provider, requests []types.ProviderRequest, collector *collector) error {
	if len(providers) == 0 || len(requests) == 0 {
		return nil
	}

	started := time.Now()
	totalJobs := len(providers) * len(requests)
	r.infof("stage %s: starting with %d provider(s), %d input(s), %d request(s)", stage, len(providers), len(requests), totalJobs)

	jobs := make(chan executionJob)
	errChan := make(chan error, r.options.Threads)

	for workerIndex := 0; workerIndex < r.options.Threads; workerIndex++ {
		go func() {
			for job := range jobs {
				if ctx.Err() != nil {
					errChan <- ctx.Err()
					return
				}

				r.verbosef("stage %s: provider=%s input=%s", job.stage, job.provider.Name(), job.request.Value)

				requestStarted := time.Now()
				observations, err := job.provider.Execute(ctx, runSession, job.request)
				collector.AddStat(job.provider.Name(), job.stage, 1, len(observations), time.Since(requestStarted), err)
				if err != nil {
					collector.AddError(types.ProviderError{
						Provider: job.provider.Name(),
						Query:    job.request.Query,
						Input:    job.request.Value,
						Message:  err.Error(),
					})
					r.verbosef("stage %s: provider=%s input=%s error=%s", job.stage, job.provider.Name(), job.request.Value, err)
					continue
				}
				collector.AddObservations(observations...)
				r.verbosef("stage %s: provider=%s input=%s observations=%d", job.stage, job.provider.Name(), job.request.Value, len(observations))
			}
			errChan <- nil
		}()
	}

	for _, item := range providers {
		if item.NeedsKey() && !runSession.ProviderConfig.HasAPIKey(item.Name()) {
			collector.MarkSkipped(item.Name(), stage, "provider skipped because api_key is missing")
			collector.AddError(types.ProviderError{
				Provider: item.Name(),
				Message:  "provider skipped because api_key is missing",
			})
			r.verbosef("stage %s: provider=%s skipped because api_key is missing", stage, item.Name())
			continue
		}
		for _, request := range requests {
			jobs <- executionJob{provider: item, stage: stage, request: request}
		}
	}
	close(jobs)

	for workerIndex := 0; workerIndex < r.options.Threads; workerIndex++ {
		if err := <-errChan; err != nil && ctx.Err() != nil {
			gologger.Warning().Msgf("worker interrupted: %s", err)
			return err
		}
	}

	r.infof("stage %s: finished in %s", stage, time.Since(started).Round(time.Millisecond))
	return nil
}

func (r *Runner) info(message string) {
	if (r.options.JSON || r.options.JSONL) && !r.options.Verbose {
		return
	}
	gologger.Info().Msg(message)
}

func previewBody(body []byte, max int) string {
	text := strings.TrimSpace(string(body))
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > max {
		return text[:max] + "..."
	}
	return text
}

func (r *Runner) infof(format string, args ...interface{}) {
	if (r.options.JSON || r.options.JSONL) && !r.options.Verbose {
		return
	}
	gologger.Info().Msgf(format, args...)
}

func (r *Runner) verbosef(format string, args ...interface{}) {
	if !r.options.Verbose {
		return
	}
	gologger.Verbose().Msgf(format, args...)
}

func joinStages(stages []types.StageType) string {
	values := make([]string, 0, len(stages))
	for _, stage := range stages {
		values = append(values, string(stage))
	}
	sort.Strings(values)
	return strings.Join(values, ",")
}

type executionJob struct {
	provider sources.Provider
	stage    types.StageType
	request  types.ProviderRequest
}
