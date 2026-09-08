package types

import (
	"strings"
	"time"
)

type WorkflowType string
type StageType string
type AssetType string
type TrackingType string

type RateLimitSpec struct {
	MaxCount uint          `json:"max_count"`
	Duration time.Duration `json:"duration"`
}

const (
	WorkflowLookup  WorkflowType = "lookup"
	WorkflowHistory WorkflowType = "history"
	WorkflowReverse WorkflowType = "reverse"

	StageLookup  StageType = "lookup"
	StageHistory StageType = "history"
	StageReverse StageType = "reverse"

	AssetTypeDomain AssetType = "domain"

	TrackingTypeUA            TrackingType = "ua"
	TrackingTypeGA4           TrackingType = "ga4"
	TrackingTypeGTM           TrackingType = "gtm"
	TrackingTypeAdsense       TrackingType = "adsense"
	TrackingTypeFacebookPixel TrackingType = "facebook_pixel"
	TrackingTypeYandexMetrica TrackingType = "yandex_metrica"
	TrackingTypeAW            TrackingType = "aw"
	TrackingTypeDC            TrackingType = "dc"
	TrackingTypeGT            TrackingType = "gt"
	TrackingTypeUnknown       TrackingType = "unknown"
)

type StagePlan struct {
	Stage     StageType `json:"stage"`
	Providers []string  `json:"providers"`
}

type ExecutionPlan struct {
	Mode       string       `json:"mode"`
	Workflow   WorkflowType `json:"workflow"`
	PlanOnly   bool         `json:"plan_only"`
	Inputs     []string     `json:"inputs"`
	StagePlans []StagePlan  `json:"stages"`
}

type ProviderParams struct {
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Frequency string     `json:"frequency,omitempty"`
	Limit     int        `json:"limit"`
}

type ProviderRequest struct {
	Query        string         `json:"query"`
	Value        string         `json:"value"`
	TrackingID   string         `json:"tracking_id,omitempty"`
	TrackingType TrackingType   `json:"tracking_type,omitempty"`
	Params       ProviderParams `json:"params"`
}

type DetectedTrackingID struct {
	Type  TrackingType `json:"type"`
	Value string       `json:"value"`
}

type Observation struct {
	Input        string       `json:"input,omitempty"`
	Query        string       `json:"query"`
	Asset        string       `json:"asset"`
	AssetType    AssetType    `json:"asset_type"`
	TrackingID   string       `json:"tracking_id"`
	TrackingType TrackingType `json:"tracking_type"`
	Source       string       `json:"source"`
	Stage        StageType    `json:"stage"`
	FirstSeen    *time.Time   `json:"first_seen,omitempty"`
	LastSeen     *time.Time   `json:"last_seen,omitempty"`
	Current      bool         `json:"current"`
	Historical   bool         `json:"historical"`
	Evidence     string       `json:"evidence,omitempty"`
}

type Finding struct {
	Inputs       []string     `json:"inputs,omitempty"`
	Queries      []string     `json:"queries"`
	Asset        string       `json:"asset"`
	AssetType    AssetType    `json:"asset_type"`
	TrackingID   string       `json:"tracking_id"`
	TrackingType TrackingType `json:"tracking_type"`
	Sources      []string     `json:"sources"`
	Stages       []StageType  `json:"stages"`
	FirstSeen    *time.Time   `json:"first_seen,omitempty"`
	LastSeen     *time.Time   `json:"last_seen,omitempty"`
	Current      bool         `json:"current"`
	Historical   bool         `json:"historical"`
	Evidence     []string     `json:"evidence,omitempty"`
}

type ProviderError struct {
	Provider string `json:"provider"`
	Query    string `json:"query,omitempty"`
	Input    string `json:"input,omitempty"`
	Message  string `json:"message"`
}

type ProviderStat struct {
	Provider       string        `json:"provider"`
	Stage          StageType     `json:"stage"`
	Requests       int           `json:"requests"`
	Observations   int           `json:"observations"`
	Findings       int           `json:"findings"`
	Errors         int           `json:"errors"`
	Skipped        bool          `json:"skipped"`
	SkipReason     string        `json:"skip_reason,omitempty"`
	Duration       time.Duration `json:"duration"`
	AverageLatency time.Duration `json:"average_latency"`
}

type ExecutionStats struct {
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt time.Time      `json:"finished_at"`
	Duration   time.Duration  `json:"duration"`
	Providers  []ProviderStat `json:"providers,omitempty"`
}

type ExecutionResult struct {
	Plan     *ExecutionPlan  `json:"plan"`
	Findings []Finding       `json:"findings"`
	Errors   []ProviderError `json:"errors,omitempty"`
	Stats    ExecutionStats  `json:"stats"`
}

func ClassifyTrackingID(value string) TrackingType {
	value = strings.ToUpper(strings.TrimSpace(value))

	switch {
	case strings.HasPrefix(value, "UA-"):
		return TrackingTypeUA
	case strings.HasPrefix(value, "GTM-"):
		return TrackingTypeGTM
	case strings.HasPrefix(value, "G-"):
		return TrackingTypeGA4
	case strings.HasPrefix(value, "CA-PUB-"):
		return TrackingTypeAdsense
	case strings.HasPrefix(value, "PUB-"):
		return TrackingTypeAdsense
	case strings.HasPrefix(value, "FBP-"):
		return TrackingTypeFacebookPixel
	case strings.HasPrefix(value, "YM-"):
		return TrackingTypeYandexMetrica
	case strings.HasPrefix(value, "AW-"):
		return TrackingTypeAW
	case strings.HasPrefix(value, "DC-"):
		return TrackingTypeDC
	case strings.HasPrefix(value, "GT-"):
		return TrackingTypeGT
	default:
		return TrackingTypeUnknown
	}
}

func TrackingProvider(value TrackingType) string {
	switch value {
	case TrackingTypeUA, TrackingTypeGA4, TrackingTypeGTM, TrackingTypeAdsense, TrackingTypeAW, TrackingTypeDC, TrackingTypeGT:
		return "google"
	case TrackingTypeFacebookPixel:
		return "facebook"
	case TrackingTypeYandexMetrica:
		return "yandex"
	default:
		return "unknown"
	}
}
