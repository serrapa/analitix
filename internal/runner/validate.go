package runner

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/projectdiscovery/goflags"
)

func validateOptions(opts *Options) error {
	if opts.Mode != ModeLookup && opts.Mode != ModeHistory && opts.Mode != ModeReverse {
		return fmt.Errorf("unsupported mode %q", opts.Mode)
	}
	if opts.ReverseDepth != ReverseDepthLookup && opts.ReverseDepth != ReverseDepthLookupHistory {
		return fmt.Errorf("unsupported reverse depth %q (supported: %s, %s)", opts.ReverseDepth, ReverseDepthLookup, ReverseDepthLookupHistory)
	}

	hasURLs := len(opts.URLs) > 0
	hasTrackingIDs := len(opts.TrackingIDs) > 0
	if !opts.ListSources && !opts.Update && !hasURLs && !hasTrackingIDs {
		return errors.New("provide at least one input with -u/-uL or -tracking-id/-tracking-id-list")
	}

	if hasURLs && hasTrackingIDs {
		return errors.New("mixed URL/domain and tracking-ID inputs are not supported in the same run")
	}

	if hasTrackingIDs && opts.Mode != ModeReverse {
		return errors.New("tracking-ID input is only valid with -m reverse")
	}
	if opts.JSON && opts.JSONL {
		return errors.New("-json and -jsonl are mutually exclusive")
	}

	if opts.Threads <= 0 {
		return errors.New("-threads must be greater than 0")
	}
	if opts.Retries <= 0 {
		return errors.New("-retry must be greater than 0")
	}
	if opts.Delay < 0 {
		return errors.New("-delay cannot be negative")
	}

	if strings.TrimSpace(opts.Proxy) != "" {
		proxyURL, err := url.Parse(strings.TrimSpace(opts.Proxy))
		if err != nil {
			return fmt.Errorf("invalid proxy URL: %w", err)
		}
		if proxyURL.Scheme == "" || proxyURL.Host == "" {
			return fmt.Errorf("invalid proxy URL %q", opts.Proxy)
		}
	}

	return nil
}

func appendFileLines(base goflags.StringSlice, path string) (goflags.StringSlice, error) {
	if strings.TrimSpace(path) == "" {
		return base, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %q: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		base = append(base, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not read %q: %w", path, err)
	}

	return base, nil
}
