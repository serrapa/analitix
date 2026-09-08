package providerconfig

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Providers map[string]ProviderSettings `yaml:",inline"`
}

type ProviderSettings struct {
	APIKey string `yaml:"api_key"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Providers: make(map[string]ProviderSettings),
	}

	if strings.TrimSpace(path) == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderSettings)
	}

	return cfg, nil
}

func (c *Config) APIKey(provider string) string {
	settings, ok := c.Providers[provider]
	if ok {
		if value := strings.TrimSpace(settings.APIKey); value != "" {
			return value
		}
	}

	providerEnv := strings.ToUpper(strings.ReplaceAll(provider, "-", "_")) + "_API_KEY"
	if value := strings.TrimSpace(os.Getenv(providerEnv)); value != "" {
		return value
	}

	return ""
}

func (c *Config) HasAPIKey(provider string) bool {
	return c.APIKey(provider) != ""
}
