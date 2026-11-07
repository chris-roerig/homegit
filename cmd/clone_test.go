package cmd

import (
	"testing"

	"github.com/chris-roerig/homegit/internal/config"
)

// TestCloneRoleSelection verifies that clone's local-vs-remote decision is
// driven by the client configuration via IsRemoteClient.
func TestCloneRoleSelection(t *testing.T) {
	tests := []struct {
		name       string
		clientHost string
		wantRemote bool
	}{
		{"no client host -> local", "", false},
		{"localhost -> local", "localhost", false},
		{"loopback -> local", "127.0.0.1", false},
		{"remote ip -> remote", "10.0.0.142", true},
		{"remote host -> remote", "house.local", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Client.Host = tt.clientHost
			if got := cfg.IsRemoteClient(); got != tt.wantRemote {
				t.Errorf("IsRemoteClient() with host %q = %v, want %v", tt.clientHost, got, tt.wantRemote)
			}
		})
	}
}
