package runner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/projectdiscovery/goflags"
)

const (
	ModeLookup  = "lookup"
	ModeHistory = "history"
	ModeReverse = "reverse"

	ReverseDepthLookup        = "lookup"
	ReverseDepthLookupHistory = "lookup-history"
)

type Options struct {
	URLs           goflags.StringSlice
	URLList        string
	TrackingIDs    goflags.StringSlice
	TrackingIDList string

	Mode string
	// ReverseDepth controls stage depth for reverse mode when starting from URL/domain input.
	// Supported values: lookup (default), lookup-history.
	ReverseDepth string

	Sources        goflags.StringSlice
	ExcludeSources goflags.StringSlice
	ListSources    bool

	StartDate string
	EndDate   string
	Frequency string
	Limit     int

	Output   string
	JSON     bool
	JSONL    bool
	PlanOnly bool
	Stats    bool
	Silent   bool
	Verbose  bool
	NoColor  bool
	Version  bool
	Timeout  int
	Threads  int
	Proxy    string
	Delay    int
	Retries  int

	RateLimit  int
	RateLimits goflags.RateLimitMap

	Config             string
	ProviderConfig     string
	DisableUpdateCheck bool
	Update             bool
}

func ParseOptions(args []string) (*Options, error) {
	opts := defaultOptions()
	configPath := configPathFromArgs(args, opts.Config)
	if err := loadConfigIntoOptions(configPath, opts); err != nil {
		return nil, fmt.Errorf("could not load config file %q: %w", configPath, err)
	}
	if configPath != "" {
		opts.Config = configPath
	}

	defaultURLs := append(goflags.StringSlice(nil), opts.URLs...)
	defaultTrackingIDs := append(goflags.StringSlice(nil), opts.TrackingIDs...)
	defaultSources := append(goflags.StringSlice(nil), opts.Sources...)
	defaultExcludeSources := append(goflags.StringSlice(nil), opts.ExcludeSources...)
	opts.URLs = nil
	opts.TrackingIDs = nil
	opts.Sources = nil
	opts.ExcludeSources = nil

	flagSet := goflags.NewFlagSet()
	flagSet.SetDescription("analytics and tracking-code OSINT")

	flagSet.CreateGroup("input", "Input",
		flagSet.StringSliceVarP(&opts.URLs, "url", "u", defaultURLs, "target URL(s) or domain(s)", goflags.CommaSeparatedStringSliceOptions),
		flagSet.StringVarP(&opts.URLList, "list", "uL", opts.URLList, "file containing target URL(s) or domain(s)"),
		flagSet.StringSliceVar(&opts.TrackingIDs, "tracking-id", defaultTrackingIDs, "tracking ID(s) for direct reverse lookup", goflags.CommaSeparatedStringSliceOptions),
		flagSet.StringVar(&opts.TrackingIDList, "tracking-id-list", opts.TrackingIDList, "file containing tracking ID(s)"),
	)

	flagSet.CreateGroup("mode", "Mode",
		flagSet.StringVarP(&opts.Mode, "mode", "m", opts.Mode, "execution mode: lookup, history, or reverse"),
		flagSet.StringVar(&opts.ReverseDepth, "reverse-depth", opts.ReverseDepth, "reverse depth for URL/domain input: lookup (default) or lookup-history"),
	)

	flagSet.CreateGroup("source", "Source",
		flagSet.StringSliceVarP(&opts.Sources, "sources", "s", defaultSources, "provider(s) to include", goflags.CommaSeparatedStringSliceOptions),
		flagSet.StringSliceVarP(&opts.ExcludeSources, "exclude-sources", "es", defaultExcludeSources, "provider(s) to exclude", goflags.CommaSeparatedStringSliceOptions),
		flagSet.BoolVarP(&opts.ListSources, "list-sources", "ls", opts.ListSources, "list available providers"),
	)

	flagSet.CreateGroup("output", "Output",
		flagSet.StringVarP(&opts.Output, "output", "o", opts.Output, "write output to file"),
		flagSet.BoolVarP(&opts.JSON, "json", "oJ", opts.JSON, "write JSON instead of text"),
		flagSet.BoolVarP(&opts.JSONL, "jsonl", "oJL", opts.JSONL, "write JSONL instead of text"),
		flagSet.BoolVar(&opts.PlanOnly, "plan", opts.PlanOnly, "print the execution plan only"),
		flagSet.BoolVar(&opts.Stats, "stats", opts.Stats, "display per-provider execution statistics"),
	)

	flagSet.CreateGroup("optimization", "Optimization",
		flagSet.IntVarP(&opts.Timeout, "timeout", "rt", opts.Timeout, "request timeout in seconds"),
		flagSet.IntVarP(&opts.Threads, "threads", "t", opts.Threads, "number of concurrent workers"),
	)

	flagSet.CreateGroup("rate-limit", "Rate Limit",
		flagSet.IntVarP(&opts.RateLimit, "rate-limit", "rl", opts.RateLimit, "global request rate limit per second"),
		flagSet.RateLimitMapVarP(&opts.RateLimits, "rate-limits", "rls", nil, "per-provider rate limits (example: hackertarget=1/s,wayback=1/s)", goflags.StringSliceOptions),
		flagSet.IntVarP(&opts.Delay, "delay", "rd", opts.Delay, "delay between outbound requests in seconds"),
		flagSet.IntVarP(&opts.Retries, "retry", "rr", opts.Retries, "number of attempts for outbound requests"),
	)

	flagSet.CreateGroup("debug", "Debug",
		flagSet.BoolVar(&opts.Silent, "silent", opts.Silent, "display results only"),
		flagSet.BoolVarP(&opts.Verbose, "verbose", "v", opts.Verbose, "display verbose output"),
		flagSet.BoolVarP(&opts.NoColor, "no-color", "nc", opts.NoColor, "disable color output"),
		flagSet.StringVar(&opts.Proxy, "proxy", opts.Proxy, "proxy URL for outbound requests (http://, https://, socks5://)"),
		flagSet.BoolVar(&opts.Version, "version", opts.Version, "show version"),
	)

	flagSet.CreateGroup("history", "History",
		flagSet.StringVar(&opts.StartDate, "start-date", opts.StartDate, "start date for historical provider queries"),
		flagSet.StringVar(&opts.EndDate, "end-date", opts.EndDate, "end date for historical provider queries"),
		flagSet.StringVarP(&opts.Frequency, "frequency", "f", opts.Frequency, "dedupe frequency for repeated historical matches"),
		flagSet.IntVarP(&opts.Limit, "limit", "l", opts.Limit, "maximum historical records per provider"),
	)

	flagSet.CreateGroup("configuration", "Configuration",
		flagSet.StringVar(&opts.Config, "config", opts.Config, "config file path"),
		flagSet.StringVarP(&opts.ProviderConfig, "provider-config", "pc", opts.ProviderConfig, "provider config file path"),
	)

	flagSet.CreateGroup("update", "Update",
		flagSet.BoolVarP(&opts.Update, "update", "up", opts.Update, "update analitix to the latest version placeholder"),
		flagSet.BoolVarP(&opts.DisableUpdateCheck, "disable-update-check", "duc", opts.DisableUpdateCheck, "disable automatic update checks placeholder"),
	)

	if err := flagSet.Parse(args...); err != nil {
		return nil, err
	}

	if err := hydrateInputs(opts); err != nil {
		return nil, err
	}

	normalizeOptions(opts)

	if err := validateOptions(opts); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return opts, nil
}

func defaultOptions() *Options {
	configRoot, err := os.UserConfigDir()
	if err != nil {
		configRoot = "."
	}
	baseDir := filepath.Join(configRoot, "analitix")

	return &Options{
		Mode:           ModeLookup,
		ReverseDepth:   ReverseDepthLookup,
		Limit:          -100,
		Timeout:        60,
		Threads:        10,
		Retries:        1,
		Config:         filepath.Join(baseDir, "config.yaml"),
		ProviderConfig: filepath.Join(baseDir, "provider-config.yaml"),
	}
}

func hydrateInputs(opts *Options) error {
	var err error
	opts.URLs, err = appendFileLines(opts.URLs, opts.URLList)
	if err != nil {
		return err
	}

	opts.TrackingIDs, err = appendFileLines(opts.TrackingIDs, opts.TrackingIDList)
	if err != nil {
		return err
	}

	return nil
}

func normalizeOptions(opts *Options) {
	if len(opts.TrackingIDs) > 0 && opts.Mode == ModeLookup {
		opts.Mode = ModeReverse
	}
}
