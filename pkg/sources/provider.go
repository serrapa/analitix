package sources

import (
	"context"

	"github.com/serrapa/analitix/pkg/session"
	"github.com/serrapa/analitix/pkg/types"
)

type Provider interface {
	Name() string
	IsDefault() bool
	NeedsKey() bool
	DefaultRateLimit() *types.RateLimitSpec
	Stages() []types.StageType
	SupportsStage(stage types.StageType) bool
	Execute(ctx context.Context, session *session.Client, request types.ProviderRequest) ([]types.Observation, error)
}

type Selection struct {
	Workflow         types.WorkflowType
	Include          []string
	Exclude          []string
	DirectTrackingID bool
}
