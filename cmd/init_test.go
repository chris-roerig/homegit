package cmd

import "testing"

func TestNormalizeRepoName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Foo Bar", "foo_bar"},
		{"foo_bar", "foo_bar"},
		{"  My-App. ", "my_app"},
		{"already-good", "already_good"},
		{"MixedCASE", "mixedcase"},
		{"multiple   spaces", "multiple_spaces"},
		{"dots.and-dashes", "dots_and_dashes"},
		{"trailing___underscores___", "trailing_underscores"},
		{"weird!@#chars$name", "weirdcharsname"},
		{"café déjà", "caf_dj"}, // non-ascii letters dropped
		{"123-abc", "123_abc"},
		{"...", ""},
		{"   ", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := normalizeRepoName(tt.in); got != tt.want {
				t.Errorf("normalizeRepoName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
