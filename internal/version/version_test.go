package version

import (
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.0.0", "1.0.0"},
		{"  v1.1.0  ", "1.1.0"},
		{"v2.0.0-beta", "2.0.0-beta"},
	}
	for _, tt := range tests {
		got := NormalizeVersion(tt.in)
		if got != tt.want {
			t.Errorf("NormalizeVersion(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestLess(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want bool
	}{
		{"1.0.0", "1.1.0", true},
		{"v1.0.0", "v1.1.0", true},
		{"1.1.0", "1.0.0", false},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "2.0.0", true},
		{"1.9.0", "2.0.0", true},
	}
	for _, tt := range tests {
		got := Less(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("Less(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestAssetNameFor(t *testing.T) {
	// Result depends on runtime.GOOS/GOARCH; at least check format
	got := AssetNameFor("v1.1.0")
	if got == "" {
		t.Error("AssetNameFor returned empty")
	}
	// Should start with prefix and contain tag
	if len(got) < 14 {
		t.Errorf("AssetNameFor too short: %q", got)
	}
	if got[:8] != "gocloud-" || got[8:14] != "v1.1.0" {
		t.Errorf("AssetNameFor should start with gocloud-v1.1.0..., got %q", got)
	}
}
