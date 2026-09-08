package runner

import (
	"testing"

	"github.com/projectdiscovery/goflags"
	"github.com/serrapa/analitix/pkg/types"
)

func TestBuildPlanLookupWorkflow(t *testing.T) {
	options := defaultOptions()
	options.URLs = goflags.StringSlice{"https://example.com"}
	options.PlanOnly = true

	r, err := New(options)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	plan, err := r.BuildPlan()
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if plan.Workflow != types.WorkflowLookup {
		t.Fatalf("unexpected workflow: %s", plan.Workflow)
	}

	if len(plan.StagePlans) != 1 || plan.StagePlans[0].Stage != types.StageLookup {
		t.Fatalf("unexpected stage plans: %+v", plan.StagePlans)
	}
}

func TestBuildPlanHistoryWorkflow(t *testing.T) {
	options := defaultOptions()
	options.Mode = ModeHistory
	options.URLs = goflags.StringSlice{"https://example.com"}
	options.PlanOnly = true

	r, err := New(options)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	plan, err := r.BuildPlan()
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if plan.Workflow != types.WorkflowHistory {
		t.Fatalf("unexpected workflow: %s", plan.Workflow)
	}

	if len(plan.StagePlans) != 1 || plan.StagePlans[0].Stage != types.StageHistory {
		t.Fatalf("unexpected stage plans: %+v", plan.StagePlans)
	}
}

func TestBuildPlanReverseWorkflow(t *testing.T) {
	options := defaultOptions()
	options.Mode = ModeReverse
	options.URLs = goflags.StringSlice{"https://example.com"}
	options.PlanOnly = true

	r, err := New(options)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	plan, err := r.BuildPlan()
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if plan.Workflow != types.WorkflowReverse {
		t.Fatalf("unexpected workflow: %s", plan.Workflow)
	}

	if len(plan.StagePlans) != 2 {
		t.Fatalf("unexpected stage count: %d", len(plan.StagePlans))
	}
}

func TestBuildPlanReverseWorkflowLookupHistoryDepth(t *testing.T) {
	options := defaultOptions()
	options.Mode = ModeReverse
	options.ReverseDepth = ReverseDepthLookupHistory
	options.URLs = goflags.StringSlice{"https://example.com"}
	options.PlanOnly = true

	r, err := New(options)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	plan, err := r.BuildPlan()
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if plan.Workflow != types.WorkflowReverse {
		t.Fatalf("unexpected workflow: %s", plan.Workflow)
	}

	if len(plan.StagePlans) != 3 {
		t.Fatalf("unexpected stage count: %d", len(plan.StagePlans))
	}
}

func TestBuildPlanTrackingIDWorkflow(t *testing.T) {
	options := defaultOptions()
	options.Mode = ModeReverse
	options.TrackingIDs = goflags.StringSlice{"UA-12345-1"}
	options.PlanOnly = true

	r, err := New(options)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	plan, err := r.BuildPlan()
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}

	if plan.Workflow != types.WorkflowReverse {
		t.Fatalf("unexpected workflow: %s", plan.Workflow)
	}

	if len(plan.StagePlans) != 1 || plan.StagePlans[0].Stage != types.StageReverse {
		t.Fatalf("unexpected stage plans: %+v", plan.StagePlans)
	}
}

func TestParseDateTime(t *testing.T) {
	value, err := parseDateTime("02/01/2024:15:04")
	if err != nil {
		t.Fatalf("parseDateTime returned error: %v", err)
	}
	if value == nil || value.Year() != 2024 || value.Month() != 1 || value.Day() != 2 {
		t.Fatalf("unexpected time value: %v", value)
	}
}
