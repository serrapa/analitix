package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/serrapa/analitix/pkg/types"
)

func WritePlan(writer io.Writer, plan *types.ExecutionPlan, asJSON bool) error {
	if asJSON {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(plan)
	}

	if _, err := io.WriteString(writer, "Execution Plan\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "workflow : %s\n", plan.Workflow); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "mode     : %s\n", plan.Mode); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "inputs   : %s\n", strings.Join(plan.Inputs, ", ")); err != nil {
		return err
	}
	if _, err := io.WriteString(writer, "stages:\n"); err != nil {
		return err
	}
	for _, stage := range plan.StagePlans {
		if _, err := fmt.Fprintf(writer, "- %s: %s\n", stage.Stage, strings.Join(stage.Providers, ", ")); err != nil {
			return err
		}
	}
	return nil
}

func WriteExecutionResult(writer io.Writer, result *types.ExecutionResult, asJSON bool, asJSONL bool, includeStats bool) error {
	if asJSON {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}
	if asJSONL {
		return writeExecutionResultJSONL(writer, result)
	}

	if _, err := io.WriteString(writer, "Summary\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "workflow : %s\n", result.Plan.Workflow); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "inputs   : %d\n", len(result.Plan.Inputs)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "findings : %d\n", len(result.Findings)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "errors   : %d\n", len(result.Errors)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "duration : %s\n", result.Stats.Duration.Round(1e6)); err != nil {
		return err
	}

	if _, err := io.WriteString(writer, "\nFindings\n"); err != nil {
		return err
	}
	if len(result.Findings) == 0 {
		if _, err := io.WriteString(writer, "- none\n"); err != nil {
			return err
		}
	} else {
		for _, finding := range result.Findings {
			firstSeen := "-"
			lastSeen := "-"
			if finding.FirstSeen != nil {
				firstSeen = finding.FirstSeen.Format(timeLayout)
			}
			if finding.LastSeen != nil {
				lastSeen = finding.LastSeen.Format(timeLayout)
			}
			line := fmt.Sprintf(
				"- asset=%s tracking_id=%s type=%s current=%t historical=%t first_seen=%s last_seen=%s sources=%s queries=%s\n",
				finding.Asset,
				finding.TrackingID,
				finding.TrackingType,
				finding.Current,
				finding.Historical,
				firstSeen,
				lastSeen,
				joinOrDash(finding.Sources),
				joinOrDash(finding.Queries),
			)
			if _, err := io.WriteString(writer, line); err != nil {
				return err
			}
		}
	}

	if len(result.Errors) > 0 {
		if _, err := io.WriteString(writer, "\nErrors\n"); err != nil {
			return err
		}
		for _, item := range result.Errors {
			line := fmt.Sprintf("- provider=%s", item.Provider)
			if item.Query != "" {
				line += " query=" + item.Query
			}
			if item.Input != "" {
				line += " input=" + item.Input
			}
			line += " message=" + item.Message + "\n"
			if _, err := io.WriteString(writer, line); err != nil {
				return err
			}
		}
	}

	if includeStats {
		if _, err := io.WriteString(writer, "\nProvider Stats\n"); err != nil {
			return err
		}
		if len(result.Stats.Providers) == 0 {
			if _, err := io.WriteString(writer, "- none\n"); err != nil {
				return err
			}
		} else {
			table := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
			if _, err := fmt.Fprintln(table, "provider\tstage\trequests\tobservations\tfindings\terrors\tskipped\tduration\tavg\tskip_reason"); err != nil {
				return err
			}
			for _, stat := range result.Stats.Providers {
				if _, err := fmt.Fprintf(
					table,
					"%s\t%s\t%d\t%d\t%d\t%d\t%t\t%s\t%s\t%s\n",
					stat.Provider,
					stat.Stage,
					stat.Requests,
					stat.Observations,
					stat.Findings,
					stat.Errors,
					stat.Skipped,
					stat.Duration.Round(1e6),
					stat.AverageLatency.Round(1e6),
					stat.SkipReason,
				); err != nil {
					return err
				}
			}
			if err := table.Flush(); err != nil {
				return err
			}
		}
	}

	return nil
}

type executionResultJSONLRecord struct {
	Input            string             `json:"input"`
	Mode             string             `json:"mode"`
	TrackingID       string             `json:"tracking_id,omitempty"`
	TrackingType     types.TrackingType `json:"tracking_type,omitempty"`
	TrackingProvider string             `json:"tracking_provider,omitempty"`
	InputType        types.TrackingType `json:"input_type,omitempty"`
	InputProvider    string             `json:"input_provider,omitempty"`
	Domains          []string           `json:"domains,omitempty"`
	IPs              []string           `json:"ips,omitempty"`
}

type lookupJSONLRecord struct {
	Input            string             `json:"input"`
	Mode             string             `json:"mode"`
	TrackingID       string             `json:"tracking_id"`
	TrackingType     types.TrackingType `json:"tracking_type"`
	TrackingProvider string             `json:"tracking_provider"`
}

type reverseJSONLRecord struct {
	Input         string             `json:"input"`
	Mode          string             `json:"mode"`
	InputType     types.TrackingType `json:"input_type"`
	InputProvider string             `json:"input_provider"`
	Domain        string             `json:"domain,omitempty"`
	IP            string             `json:"ip,omitempty"`
	Source        string             `json:"source,omitempty"`
	Timestamp     *string            `json:"timestamp,omitempty"`
}

func writeExecutionResultJSONL(writer io.Writer, result *types.ExecutionResult) error {
	encoder := json.NewEncoder(writer)
	mode := strings.TrimSpace(strings.ToLower(result.Plan.Mode))
	switch mode {
	case "lookup":
		return writeLookupJSONL(encoder, result)
	case "reverse":
		return writeReverseJSONL(encoder, result)
	default:
		return writeGenericJSONL(encoder, result)
	}
}

func writeLookupJSONL(encoder *json.Encoder, result *types.ExecutionResult) error {
	for _, finding := range result.Findings {
		inputs := finding.Inputs
		if len(inputs) == 0 {
			inputs = append([]string(nil), finding.Queries...)
		}
		if len(inputs) == 0 {
			inputs = append([]string(nil), result.Plan.Inputs...)
		}
		if len(inputs) == 0 {
			inputs = []string{""}
		}

		for _, input := range inputs {
			record := lookupJSONLRecord{
				Input:            input,
				Mode:             "lookup",
				TrackingID:       finding.TrackingID,
				TrackingType:     finding.TrackingType,
				TrackingProvider: types.TrackingProvider(finding.TrackingType),
			}
			if err := encoder.Encode(record); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeReverseJSONL(encoder *json.Encoder, result *types.ExecutionResult) error {
	for _, finding := range result.Findings {
		if !containsStage(finding.Stages, types.StageReverse) {
			continue
		}
		input := strings.TrimSpace(finding.TrackingID)
		if input == "" {
			continue
		}

		domain, ip := splitFindingAsset(finding)
		sources := append([]string(nil), finding.Sources...)
		sort.Strings(sources)
		if len(sources) == 0 {
			sources = []string{""}
		}

		timestamp := reverseTimestamp(finding)
		for _, source := range sources {
			record := reverseJSONLRecord{
				Input:         input,
				Mode:          "reverse",
				InputType:     finding.TrackingType,
				InputProvider: types.TrackingProvider(finding.TrackingType),
				Domain:        domain,
				IP:            ip,
				Source:        source,
				Timestamp:     timestamp,
			}
			if err := encoder.Encode(record); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeGenericJSONL(encoder *json.Encoder, result *types.ExecutionResult) error {
	for _, finding := range result.Findings {
		inputs := finding.Inputs
		if len(inputs) == 0 {
			inputs = append([]string(nil), finding.Queries...)
		}
		if len(inputs) == 0 {
			inputs = append([]string(nil), result.Plan.Inputs...)
		}
		if len(inputs) == 0 {
			inputs = []string{""}
		}

		domains, ips := splitFindingAssets(finding)
		for _, input := range inputs {
			record := executionResultJSONLRecord{
				Input:            input,
				Mode:             strings.TrimSpace(strings.ToLower(result.Plan.Mode)),
				TrackingID:       finding.TrackingID,
				TrackingType:     finding.TrackingType,
				TrackingProvider: types.TrackingProvider(finding.TrackingType),
				Domains:          domains,
				IPs:              ips,
			}
			if err := encoder.Encode(record); err != nil {
				return err
			}
		}
	}
	return nil
}

func splitFindingAssets(finding types.Finding) ([]string, []string) {
	switch finding.AssetType {
	case types.AssetTypeDomain:
		if strings.TrimSpace(finding.Asset) == "" {
			return []string{}, []string{}
		}
		return []string{finding.Asset}, []string{}
	default:
		if strings.TrimSpace(finding.Asset) == "" {
			return []string{}, []string{}
		}
		return []string{}, []string{finding.Asset}
	}
}

func splitFindingAsset(finding types.Finding) (string, string) {
	domains, ips := splitFindingAssets(finding)
	var domain string
	var ip string
	if len(domains) > 0 {
		domain = domains[0]
	}
	if len(ips) > 0 {
		ip = ips[0]
	}
	return domain, ip
}

func containsStage(stages []types.StageType, target types.StageType) bool {
	for _, stage := range stages {
		if stage == target {
			return true
		}
	}
	return false
}

func sortedKeys(values map[string]struct{}) []string {
	if len(values) == 0 {
		return []string{}
	}
	items := make([]string, 0, len(values))
	for key := range values {
		items = append(items, key)
	}
	sort.Strings(items)
	return items
}

func reverseTimestamp(finding types.Finding) *string {
	switch {
	case finding.LastSeen != nil:
		value := finding.LastSeen.Format(timeLayout)
		return &value
	case finding.FirstSeen != nil:
		value := finding.FirstSeen.Format(timeLayout)
		return &value
	default:
		return nil
	}
}

func joinOrDash(values []string) string {
	if len(values) == 0 {
		return "-"
	}
	return strings.Join(values, ",")
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func Open(path string, stdout io.Writer) (io.Writer, func() error, error) {
	if strings.TrimSpace(path) == "" {
		return stdout, func() error { return nil }, nil
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}

	return file, file.Close, nil
}
