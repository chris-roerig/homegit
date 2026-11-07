package cmd

import (
	"strings"
	"testing"
)

func TestVersionLabel(t *testing.T) {
	label := versionLabel()
	if !strings.HasPrefix(label, "homegit ") {
		t.Errorf("label should start with 'homegit ', got %q", label)
	}
	if !strings.HasSuffix(label, ":") {
		t.Errorf("label should end with ':', got %q", label)
	}
	if !strings.Contains(label, Version) {
		t.Errorf("label %q should contain version %q", label, Version)
	}
}
