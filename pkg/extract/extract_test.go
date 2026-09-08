package extract

import (
	"slices"
	"testing"
)

func TestExtractParityTrackingIDs(t *testing.T) {
	body := []byte(`
        <script>
            gtag('config', 'G-ABCDE12345');
            ga('create', 'UA-12345678-1');
            googletagmanager = 'GTM-ABC1234';
            adsense = 'ca-pub-1234567890123456';
            fbq('init', '123456789012345');
            ym(12345678, 'init', {});
        </script>
    `)

	results := ExtractParityTrackingIDs(body)
	if len(results) != 6 {
		t.Fatalf("unexpected result count: %d", len(results))
	}
}

func TestExtractParityTrackingIDsSkipsGA4FalsePositives(t *testing.T) {
	body := []byte(`
        <div>
            G-FACEBOOK G-SOCIAL G-SEARCH G-USERICON
            gtag('config', 'G-1ABCDEF234');
        </div>
    `)

	results := ExtractParityTrackingIDs(body)
	values := make([]string, 0, len(results))
	for _, item := range results {
		values = append(values, item.Value)
	}

	for _, unexpected := range []string{"G-FACEBOOK", "G-SOCIAL", "G-SEARCH", "G-USERICON"} {
		if slices.Contains(values, unexpected) {
			t.Fatalf("unexpected false positive GA4 id found: %s", unexpected)
		}
	}
	if !slices.Contains(values, "G-1ABCDEF234") {
		t.Fatalf("expected valid GA4 id not found")
	}
}
