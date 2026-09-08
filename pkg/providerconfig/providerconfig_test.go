package providerconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAPIKeyPrefersConfigOverEnvironment(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "provider-config.yaml")
	config := "hackertarget:\n  api_key: from-config\n"
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("write provider config: %v", err)
	}

	t.Setenv("HACKERTARGET_API_KEY", "from-env")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load provider config: %v", err)
	}

	if value := cfg.APIKey("hackertarget"); value != "from-config" {
		t.Fatalf("expected config api key precedence, got %q", value)
	}
}

func TestAPIKeyFallsBackToGenericEnvironmentVariable(t *testing.T) {
	cfg := &Config{Providers: map[string]ProviderSettings{}}
	t.Setenv("OSINTSH_API_KEY", "from-env")

	if value := cfg.APIKey("osintsh"); value != "from-env" {
		t.Fatalf("expected generic env api key, got %q", value)
	}
}
