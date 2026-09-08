package targets

import "testing"

func TestAssetFromInputRejectsInvalid(t *testing.T) {
	if got := AssetFromInput("invalid value"); got != "" {
		t.Fatalf("expected invalid asset to be rejected, got %q", got)
	}
}

func TestAssetFromInputNormalizesURL(t *testing.T) {
	if got := AssetFromInput("https://WWW.Example.com/path"); got != "www.example.com" {
		t.Fatalf("unexpected normalized asset: %q", got)
	}
}
