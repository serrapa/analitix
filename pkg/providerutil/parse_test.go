package providerutil

import "testing"

func TestNormalizeDomains(t *testing.T) {
	inputs := []string{"https://www.example.com/path", "example.com", "invalid value"}
	got := NormalizeDomains(inputs)
	if len(got) != 2 {
		t.Fatalf("unexpected normalized domain count: %d", len(got))
	}
}
