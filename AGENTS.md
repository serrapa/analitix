# AGENTS.md

Operational reference for AI agents working in this repository. Read this before making structural or architectural changes.

## What This Tool Is

`analitix` is a Go CLI for OSINT on analytics/tracking codes (Google UA/GA4, GTM, AdSense, Facebook Pixel, Yandex Metrica, etc.) tied to domains and URLs. Three modes:

- **lookup** — extract tracking IDs live from a target URL/domain.
- **history** — extract tracking IDs seen historically (e.g. via Wayback Machine).
- **reverse** — given a tracking ID, find other domains that share it (pivot from an identifier to related infrastructure).

Go module: `github.com/serrapa/analitix`. Never use a different or placeholder module path in imports.

## Repository Structure

```text
cmd/analitix/main.go       bootstrap only: parse options, run the runner
internal/runner/
  options.go               flag definitions, defaults, config loading
  validate.go               option validation
  config.go                 config file loading
  runner.go, execute.go     execution plan, orchestration, worker pool
  banner.go                 CLI banner
pkg/types/                 shared normalized contracts (ExecutionPlan, ProviderRequest,
                            Observation, Finding, ProviderError, ProviderStat,
                            WorkflowType, StageType, RateLimitSpec)
pkg/sources/
  provider.go               Provider interface + Selection type
  registry.go                Registry: lists providers, filters by workflow/selection
  wayback/, hackertarget/, osintsh/, dnslytics/   one package per provider
pkg/session/                shared HTTP client, rate limiting, proxy, retries
pkg/output/                 text/JSON/JSONL rendering
pkg/extract/                tracking-ID extraction from HTML/responses
pkg/targets/                URL/domain input normalization
pkg/providerconfig/         provider credentials/config file handling
pkg/providerutil/           small provider-facing helpers
```

Keep `cmd/analitix` minimal — no real logic there. Real logic lives in `internal/runner` (orchestration) and `pkg/*` (contracts, providers, I/O).

## Hard Rules When Extending The Code

1. **New provider = new package under `pkg/sources/`.** Implement `sources.Provider` (`Name`, `IsDefault`, `NeedsKey`, `DefaultRateLimit`, `Stages`, `SupportsStage`, `Execute`) and register it in `NewRegistry()` in [pkg/sources/registry.go](pkg/sources/registry.go).
2. **Providers stay dumb.** A provider builds requests, calls the shared `session.Client`, parses its own response, and emits `[]types.Observation`. It must not: decide which workflow is active, deduplicate across providers, or write output directly. Workflow/stage matching lives in `registry.go`'s `matchesWorkflow`, not in the provider.
3. **Missing credentials → explicit skip, not crash.** Log the reason and continue, unless the requested workflow depends exclusively on that provider.
4. **Shared contracts live in `pkg/types`.** Don't duplicate `Observation`/`Finding`-shaped structs inside a provider package; extend the shared types instead.
5. **Output discipline:** text output (via `pkg/output`) stays readable and sectioned; JSON/JSONL stays free of banners and info-level logs, with stable field names.
6. **CLI flags follow the existing groups and alias style** in [internal/runner/options.go](internal/runner/options.go) (see below) — don't invent a new naming scheme for a new flag.
7. **Don't leave placeholder behavior undocumented.** If a flag's help text says "placeholder" (currently `-up/--update` and `-duc/--disable-update-check`), treat the feature as not yet implemented — don't assume it works.

## Current CLI Flags

Defined via `goflags` groups in `options.go`. When adding a flag, put it in the matching group.

- **Input**: `-u/--url`, `--list` (`-uL`), `--tracking-id`, `--tracking-id-list`
- **Mode**: `-m/--mode` (`lookup`|`history`|`reverse`), `--reverse-depth`
- **Source**: `-s/--sources`, `-es/--exclude-sources`, `-ls/--list-sources`
- **Output**: `-o/--output`, `-oJ/--json`, `-oJL/--jsonl`, `--plan`, `--stats`
- **Optimization**: `-rt/--timeout`, `-t/--threads`
- **Rate limit**: `-rl/--rate-limit`, `-rls/--rate-limits`, `-rd/--delay`, `-rr/--retry`
- **Debug**: `--silent`, `-v/--verbose`, `-nc/--no-color`, `--proxy`, `--version`
- **History**: `--start-date`, `--end-date`, `-f/--frequency`, `-l/--limit`
- **Configuration**: `--config`, `-pc/--provider-config`
- **Update**: `-up/--update`, `-duc/--disable-update-check` (placeholders, not implemented)

## Testing Expectations

When touching code in these areas, add or update tests following the existing patterns:

- Option parsing/validation → see [internal/runner/config_test.go](internal/runner/config_test.go)
- Orchestration/planning → see [internal/runner/runner_test.go](internal/runner/runner_test.go)
- Provider parsing/behavior → see [pkg/sources/dnslytics/provider_test.go](pkg/sources/dnslytics/provider_test.go)
- Output rendering and normalization helpers should also be covered when changed.
