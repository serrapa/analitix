package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseOptionsLoadsConfigWithCLIOverride(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	config := "mode: history\nreverse-depth: lookup-history\nsources:\n  - wayback\ntimeout: 90\ndelay: 2\nretry: 3\njsonl: true\n"
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	opts, err := ParseOptions([]string{"-config", configPath, "-u", "https://example.com", "-mode", "reverse", "-timeout", "30"})
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}

	if opts.Mode != ModeReverse {
		t.Fatalf("expected cli mode override, got %q", opts.Mode)
	}
	if opts.ReverseDepth != ReverseDepthLookupHistory {
		t.Fatalf("expected reverse-depth from config, got %q", opts.ReverseDepth)
	}
	if opts.Timeout != 30 {
		t.Fatalf("expected cli timeout override, got %d", opts.Timeout)
	}
	if opts.Delay != 2 {
		t.Fatalf("expected config delay, got %d", opts.Delay)
	}
	if opts.Retries != 3 {
		t.Fatalf("expected config retry, got %d", opts.Retries)
	}
	if len(opts.Sources) != 1 || opts.Sources[0] != "wayback" {
		t.Fatalf("expected config sources, got %#v", opts.Sources)
	}
	if !opts.JSONL {
		t.Fatalf("expected config jsonl true, got false")
	}
}

func TestParseOptionsReverseDepthOverride(t *testing.T) {
	opts, err := ParseOptions([]string{"-u", "https://example.com", "-mode", "reverse", "-reverse-depth", "lookup-history"})
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}
	if opts.ReverseDepth != ReverseDepthLookupHistory {
		t.Fatalf("expected reverse depth lookup-history, got %q", opts.ReverseDepth)
	}
}

func TestParseOptionsJSONLFlag(t *testing.T) {
	opts, err := ParseOptions([]string{"-u", "https://example.com", "-jsonl"})
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}
	if !opts.JSONL {
		t.Fatalf("expected jsonl true, got false")
	}
}

func TestParseOptionsJSONAndJSONLMutuallyExclusive(t *testing.T) {
	_, err := ParseOptions([]string{"-u", "https://example.com", "-json", "-jsonl"})
	if err == nil {
		t.Fatalf("expected error for mutually exclusive flags")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("unexpected error: %v", err)
	}
}
