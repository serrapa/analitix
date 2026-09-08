package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/serrapa/analitix/pkg/types"
)

func TestWriteExecutionResultWithStats(t *testing.T) {
	started := time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC)
	finished := started.Add(1500 * time.Millisecond)
	result := &types.ExecutionResult{
		Plan: &types.ExecutionPlan{Workflow: types.WorkflowReverse, Inputs: []string{"example.com"}},
		Findings: []types.Finding{{
			Asset:        "example.com",
			TrackingID:   "UA-12345-1",
			TrackingType: types.TrackingTypeUA,
			Sources:      []string{"live"},
			Queries:      []string{"example.com"},
		}},
		Stats: types.ExecutionStats{
			StartedAt:  started,
			FinishedAt: finished,
			Duration:   finished.Sub(started),
			Providers: []types.ProviderStat{{
				Provider:       "live",
				Stage:          types.StageLookup,
				Requests:       1,
				Observations:   1,
				Findings:       1,
				Duration:       1500 * time.Millisecond,
				AverageLatency: 1500 * time.Millisecond,
			}},
		},
	}

	var buffer bytes.Buffer
	if err := WriteExecutionResult(&buffer, result, false, false, true); err != nil {
		t.Fatalf("write execution result: %v", err)
	}

	output := buffer.String()
	for _, expected := range []string{"Summary", "workflow : reverse", "Findings", "Provider Stats", "provider", "live", "lookup"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got %q", expected, output)
		}
	}
}

func TestWriteExecutionResultWithoutStats(t *testing.T) {
	result := &types.ExecutionResult{
		Plan: &types.ExecutionPlan{Workflow: types.WorkflowLookup},
		Stats: types.ExecutionStats{
			Providers: []types.ProviderStat{{Provider: "live", Stage: types.StageLookup, Requests: 1}},
		},
	}

	var buffer bytes.Buffer
	if err := WriteExecutionResult(&buffer, result, false, false, false); err != nil {
		t.Fatalf("write execution result: %v", err)
	}

	output := buffer.String()
	for _, expected := range []string{"Summary", "Findings", "- none"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got %q", expected, output)
		}
	}
	if strings.Contains(output, "Provider Stats") {
		t.Fatalf("expected output not to contain provider stats, got %q", output)
	}
}

func TestWriteExecutionResultJSONL(t *testing.T) {
	lastSeen := time.Date(2026, 3, 15, 11, 0, 0, 0, time.UTC)
	result := &types.ExecutionResult{
		Plan: &types.ExecutionPlan{Mode: "reverse", Workflow: types.WorkflowReverse, Inputs: []string{"UA-12345-1"}},
		Findings: []types.Finding{{
			Inputs:       []string{"UA-12345-1"},
			Asset:        "example.com",
			AssetType:    types.AssetTypeDomain,
			TrackingID:   "UA-12345-1",
			TrackingType: types.TrackingTypeUA,
			Sources:      []string{"dnslytics", "hackertarget"},
			Queries:      []string{"UA-12345-1"},
			Stages:       []types.StageType{types.StageReverse},
			LastSeen:     &lastSeen,
		}},
	}

	var buffer bytes.Buffer
	if err := WriteExecutionResult(&buffer, result, false, true, false); err != nil {
		t.Fatalf("write execution result jsonl: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buffer.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 jsonl rows, got %d", len(lines))
	}

	var row map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &row); err != nil {
		t.Fatalf("decode jsonl row: %v", err)
	}

	for _, key := range []string{"input", "mode", "input_type", "input_provider", "domain", "source", "timestamp"} {
		if _, ok := row[key]; !ok {
			t.Fatalf("expected key %q in jsonl row, got %v", key, row)
		}
	}
}

func TestWriteExecutionResultJSONLLookup(t *testing.T) {
	result := &types.ExecutionResult{
		Plan: &types.ExecutionPlan{Mode: "lookup", Workflow: types.WorkflowLookup, Inputs: []string{"allrentsardinia.com"}},
		Findings: []types.Finding{{
			Inputs:       []string{"allrentsardinia.com"},
			Asset:        "allrentsardinia.com",
			AssetType:    types.AssetTypeDomain,
			TrackingID:   "GTM-PS6288KW",
			TrackingType: types.TrackingTypeGTM,
		}},
	}

	var buffer bytes.Buffer
	if err := WriteExecutionResult(&buffer, result, false, true, false); err != nil {
		t.Fatalf("write execution result jsonl: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buffer.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 jsonl row, got %d", len(lines))
	}

	var row map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &row); err != nil {
		t.Fatalf("decode jsonl row: %v", err)
	}

	for _, key := range []string{"input", "mode", "tracking_id", "tracking_type", "tracking_provider"} {
		if _, ok := row[key]; !ok {
			t.Fatalf("expected key %q in jsonl row, got %v", key, row)
		}
	}
}
