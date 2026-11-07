package cmd

import (
	"testing"

	"github.com/chris-roerig/homegit/internal/config"
)

func TestIsLocalServer(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"10.0.0.142", false},
		{"example.local", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			cfg := &config.Config{ServerHost: tt.host}
			if got := isLocalServer(cfg); got != tt.want {
				t.Errorf("isLocalServer(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}
