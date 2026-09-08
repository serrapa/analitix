package extract

import (
	"regexp"
	"strings"

	"github.com/serrapa/analitix/pkg/types"
)

var plainPatterns = []struct {
	Type  types.TrackingType
	Regex *regexp.Regexp
}{
	{Type: types.TrackingTypeUA, Regex: regexp.MustCompile(`(?i)\bUA-\d{4,}-\d+\b`)},
	{Type: types.TrackingTypeGA4, Regex: regexp.MustCompile(`(?i)\bG-[A-Z0-9]{8,15}\b`)},
	{Type: types.TrackingTypeGTM, Regex: regexp.MustCompile(`(?i)\bGTM-[A-Z0-9]+\b`)},
}

var adsenseRegex = regexp.MustCompile(`(?i)\b(?:CA-)?PUB-(\d{16})\b`)

var facebookPixelInitRegex = regexp.MustCompile(`(?is)fbq\s*\(\s*['"]init['"]\s*,\s*['"]([0-9]{8,20})['"]`)
var facebookPixelDatasetRegex = regexp.MustCompile(`(?is)data-pixel-id\s*=\s*['"]([0-9]{8,20})['"]`)

var yandexMetricaInitRegex = regexp.MustCompile(`(?is)\bym\s*\(\s*([0-9]{5,20})\s*,\s*['"]init['"]`)
var yandexMetricaQueryRegex = regexp.MustCompile(`(?is)[?&]id=([0-9]{5,20})\b`)
var yandexCounterRegex = regexp.MustCompile(`(?is)\byaCounter([0-9]{5,20})\b`)

func ExtractParityTrackingIDs(body []byte) []types.DetectedTrackingID {
	text := string(body)
	seen := map[string]struct{}{}
	var detected []types.DetectedTrackingID

	for _, pattern := range plainPatterns {
		matches := pattern.Regex.FindAllString(text, -1)
		for _, match := range matches {
			match = strings.ToUpper(strings.TrimSpace(match))
			if pattern.Type == types.TrackingTypeGA4 && !looksLikeGA4(match) {
				continue
			}
			if _, ok := seen[match]; ok {
				continue
			}
			seen[match] = struct{}{}
			detected = append(detected, types.DetectedTrackingID{
				Type:  pattern.Type,
				Value: match,
			})
		}
	}

	for _, match := range adsenseRegex.FindAllStringSubmatch(text, -1) {
		if len(match) < 2 {
			continue
		}
		value := "pub-" + strings.TrimSpace(match[1])
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		detected = append(detected, types.DetectedTrackingID{
			Type:  types.TrackingTypeAdsense,
			Value: value,
		})
	}

	appendCapturedIDs(&detected, seen, types.TrackingTypeFacebookPixel, "FBP-", facebookPixelInitRegex.FindAllStringSubmatch(text, -1))
	appendCapturedIDs(&detected, seen, types.TrackingTypeFacebookPixel, "FBP-", facebookPixelDatasetRegex.FindAllStringSubmatch(text, -1))

	appendCapturedIDs(&detected, seen, types.TrackingTypeYandexMetrica, "YM-", yandexMetricaInitRegex.FindAllStringSubmatch(text, -1))
	appendCapturedIDs(&detected, seen, types.TrackingTypeYandexMetrica, "YM-", yandexMetricaQueryRegex.FindAllStringSubmatch(text, -1))
	appendCapturedIDs(&detected, seen, types.TrackingTypeYandexMetrica, "YM-", yandexCounterRegex.FindAllStringSubmatch(text, -1))

	return detected
}

func looksLikeGA4(value string) bool {
	suffix := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(value)), "G-")
	if len(suffix) < 8 || len(suffix) > 15 {
		return false
	}
	for _, char := range suffix {
		if char >= '0' && char <= '9' {
			return true
		}
	}
	return false
}

func appendCapturedIDs(detected *[]types.DetectedTrackingID, seen map[string]struct{}, trackingType types.TrackingType, prefix string, matches [][]string) {
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		id := strings.TrimSpace(match[1])
		if id == "" {
			continue
		}
		value := strings.ToUpper(prefix + id)
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		*detected = append(*detected, types.DetectedTrackingID{
			Type:  trackingType,
			Value: value,
		})
	}
}
