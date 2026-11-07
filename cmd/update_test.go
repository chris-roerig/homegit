package cmd

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"v1.2.0", "v1.1.0", 1},
		{"v1.1.0", "v1.2.0", -1},
		{"v1.2.0", "v1.2.0", 0},
		{"v1.2.1", "v1.2.0", 1},
		{"v2.0.0", "v1.9.9", 1},
		{"1.2.0", "v1.2.0", 0},      // tolerate missing 'v'
		{"v1.2.0-rc1", "v1.2.0", 0}, // pre-release suffix ignored for core compare
		{"v1.10.0", "v1.9.0", 1},    // numeric, not lexical
	}
	for _, tt := range tests {
		if got := compareVersions(tt.a, tt.b); got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsNewer(t *testing.T) {
	if !isNewer("v1.3.0", "v1.2.0") {
		t.Errorf("v1.3.0 should be newer than v1.2.0")
	}
	if isNewer("v1.2.0", "v1.2.0") {
		t.Errorf("equal versions should not be newer")
	}
	if isNewer("v1.1.0", "v1.2.0") {
		t.Errorf("older version should not be newer")
	}
}

func TestIsHomebrewInstall(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/opt/homebrew/Cellar/homegit/1.2.0/bin/homegit", true},
		{"/usr/local/Cellar/homegit/1.2.0/bin/homegit", true},
		{"/opt/homebrew/bin/homegit", true}, // under homebrew prefix
		{"/home/linuxbrew/.linuxbrew/bin/homegit", true},
		{"/usr/local/bin/homegit", false},
		{"/Users/chris/.local/bin/homegit", false},
	}
	for _, tt := range tests {
		if got := isHomebrewInstall(tt.path); got != tt.want {
			t.Errorf("isHomebrewInstall(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
