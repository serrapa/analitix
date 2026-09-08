package sources

import (
	"fmt"
	"slices"

	"github.com/serrapa/analitix/pkg/sources/dnslytics"
	"github.com/serrapa/analitix/pkg/sources/hackertarget"
	"github.com/serrapa/analitix/pkg/sources/osintsh"
	"github.com/serrapa/analitix/pkg/sources/wayback"
	"github.com/serrapa/analitix/pkg/types"
)

type Registry struct {
	all []Provider
}

func NewRegistry() *Registry {
	return &Registry{
		all: []Provider{
			wayback.New(),
			hackertarget.New(),
			osintsh.New(),
			dnslytics.New(),
		},
	}
}

func (r *Registry) All() []Provider {
	return append([]Provider(nil), r.all...)
}

func (r *Registry) DefaultRateLimits() map[string]types.RateLimitSpec {
	limits := make(map[string]types.RateLimitSpec)
	for _, item := range r.all {
		if spec := item.DefaultRateLimit(); spec != nil {
			limits[item.Name()] = *spec
		}
	}
	return limits
}

func (r *Registry) Select(selection Selection) ([]Provider, error) {
	included := map[string]struct{}{}
	for _, name := range selection.Include {
		included[name] = struct{}{}
	}

	excluded := map[string]struct{}{}
	for _, name := range selection.Exclude {
		excluded[name] = struct{}{}
	}

	var selected []Provider
	for _, item := range r.all {
		name := item.Name()
		if _, ok := excluded[name]; ok {
			continue
		}
		if len(included) > 0 {
			if _, ok := included[name]; !ok {
				continue
			}
		}

		if !matchesWorkflow(item, selection) {
			continue
		}

		selected = append(selected, item)
	}

	if len(included) > 0 {
		for name := range included {
			if !slices.ContainsFunc(selected, func(item Provider) bool { return item.Name() == name }) {
				return nil, fmt.Errorf("source %q is not available for the selected workflow", name)
			}
		}
	}

	return selected, nil
}

func matchesWorkflow(item Provider, selection Selection) bool {
	switch selection.Workflow {
	case types.WorkflowLookup:
		return item.SupportsStage(types.StageLookup)
	case types.WorkflowHistory:
		return item.SupportsStage(types.StageHistory)
	case types.WorkflowReverse:
		if selection.DirectTrackingID {
			return item.SupportsStage(types.StageReverse)
		}
		return item.SupportsStage(types.StageLookup) ||
			item.SupportsStage(types.StageHistory) ||
			item.SupportsStage(types.StageReverse)
	default:
		return false
	}
}
