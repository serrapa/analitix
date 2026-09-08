package runner

import (
	"os"
	"strings"

	"github.com/projectdiscovery/goflags"
	"gopkg.in/yaml.v3"
)

type fileConfig struct {
	URLs           []string          `yaml:"url"`
	URLList        string            `yaml:"list"`
	TrackingIDs    []string          `yaml:"tracking-id"`
	TrackingIDList string            `yaml:"tracking-id-list"`
	Mode           string            `yaml:"mode"`
	ReverseDepth   string            `yaml:"reverse-depth"`
	Sources        []string          `yaml:"sources"`
	ExcludeSources []string          `yaml:"exclude-sources"`
	ListSources    *bool             `yaml:"list-sources"`
	StartDate      string            `yaml:"start-date"`
	EndDate        string            `yaml:"end-date"`
	Frequency      string            `yaml:"frequency"`
	Limit          *int              `yaml:"limit"`
	Output         string            `yaml:"output"`
	JSON           *bool             `yaml:"json"`
	JSONL          *bool             `yaml:"jsonl"`
	PlanOnly       *bool             `yaml:"plan"`
	Stats          *bool             `yaml:"stats"`
	Silent         *bool             `yaml:"silent"`
	Verbose        *bool             `yaml:"verbose"`
	NoColor        *bool             `yaml:"no-color"`
	Version        *bool             `yaml:"version"`
	Timeout        *int              `yaml:"timeout"`
	Threads        *int              `yaml:"threads"`
	Proxy          string            `yaml:"proxy"`
	RateLimit      *int              `yaml:"rate-limit"`
	RateLimits     map[string]string `yaml:"rate-limits"`
	Delay          *int              `yaml:"delay"`
	Retries        *int              `yaml:"retry"`
	Config         string            `yaml:"config"`
	ProviderConfig string            `yaml:"provider-config"`
	Update         *bool             `yaml:"update"`
	DisableUpdate  *bool             `yaml:"disable-update-check"`
}

func loadConfigIntoOptions(path string, opts *Options) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var cfg fileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return err
	}

	if len(cfg.URLs) > 0 {
		opts.URLs = goflags.StringSlice(cfg.URLs)
	}
	if cfg.URLList != "" {
		opts.URLList = cfg.URLList
	}
	if len(cfg.TrackingIDs) > 0 {
		opts.TrackingIDs = goflags.StringSlice(cfg.TrackingIDs)
	}
	if cfg.TrackingIDList != "" {
		opts.TrackingIDList = cfg.TrackingIDList
	}
	if cfg.Mode != "" {
		opts.Mode = cfg.Mode
	}
	if cfg.ReverseDepth != "" {
		opts.ReverseDepth = cfg.ReverseDepth
	}
	if len(cfg.Sources) > 0 {
		opts.Sources = goflags.StringSlice(cfg.Sources)
	}
	if len(cfg.ExcludeSources) > 0 {
		opts.ExcludeSources = goflags.StringSlice(cfg.ExcludeSources)
	}
	if cfg.ListSources != nil {
		opts.ListSources = *cfg.ListSources
	}
	if cfg.StartDate != "" {
		opts.StartDate = cfg.StartDate
	}
	if cfg.EndDate != "" {
		opts.EndDate = cfg.EndDate
	}
	if cfg.Frequency != "" {
		opts.Frequency = cfg.Frequency
	}
	if cfg.Limit != nil {
		opts.Limit = *cfg.Limit
	}
	if cfg.Output != "" {
		opts.Output = cfg.Output
	}
	if cfg.JSON != nil {
		opts.JSON = *cfg.JSON
	}
	if cfg.JSONL != nil {
		opts.JSONL = *cfg.JSONL
	}
	if cfg.PlanOnly != nil {
		opts.PlanOnly = *cfg.PlanOnly
	}
	if cfg.Stats != nil {
		opts.Stats = *cfg.Stats
	}
	if cfg.Silent != nil {
		opts.Silent = *cfg.Silent
	}
	if cfg.Verbose != nil {
		opts.Verbose = *cfg.Verbose
	}
	if cfg.NoColor != nil {
		opts.NoColor = *cfg.NoColor
	}
	if cfg.Version != nil {
		opts.Version = *cfg.Version
	}
	if cfg.Timeout != nil {
		opts.Timeout = *cfg.Timeout
	}
	if cfg.Threads != nil {
		opts.Threads = *cfg.Threads
	}
	if cfg.Proxy != "" {
		opts.Proxy = cfg.Proxy
	}
	if cfg.RateLimit != nil {
		opts.RateLimit = *cfg.RateLimit
	}
	if len(cfg.RateLimits) > 0 {
		for name, value := range cfg.RateLimits {
			if err := opts.RateLimits.Set(name + "=" + value); err != nil {
				return err
			}
		}
	}
	if cfg.Delay != nil {
		opts.Delay = *cfg.Delay
	}
	if cfg.Retries != nil {
		opts.Retries = *cfg.Retries
	}
	if cfg.Config != "" {
		opts.Config = cfg.Config
	}
	if cfg.ProviderConfig != "" {
		opts.ProviderConfig = cfg.ProviderConfig
	}
	if cfg.Update != nil {
		opts.Update = *cfg.Update
	}
	if cfg.DisableUpdate != nil {
		opts.DisableUpdateCheck = *cfg.DisableUpdate
	}

	return nil
}

func configPathFromArgs(args []string, defaultPath string) string {
	path := strings.TrimSpace(defaultPath)
	for index := 0; index < len(args); index++ {
		item := strings.TrimSpace(args[index])
		switch {
		case item == "-config" || item == "--config":
			if index+1 < len(args) {
				path = strings.TrimSpace(args[index+1])
			}
		case strings.HasPrefix(item, "-config="):
			path = strings.TrimSpace(strings.TrimPrefix(item, "-config="))
		case strings.HasPrefix(item, "--config="):
			path = strings.TrimSpace(strings.TrimPrefix(item, "--config="))
		}
	}
	return path
}
