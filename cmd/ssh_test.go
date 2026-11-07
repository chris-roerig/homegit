package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/chris-roerig/homegit/internal/config"
)

func TestClassifySSHError(t *testing.T) {
	cfg := &config.Config{}
	cfg.Client.Host = "10.0.0.142"
	cfg.Client.Port = 2222
	runErr := errors.New("exit status 255")

	tests := []struct {
		name       string
		stderr     string
		wantSubstr string
	}{
		{
			name:       "connection refused",
			stderr:     "ssh: connect to host 10.0.0.142 port 2222: Connection refused",
			wantSubstr: "connection refused",
		},
		{
			name:       "timed out",
			stderr:     "ssh: connect to host 10.0.0.142 port 2222: Operation timed out",
			wantSubstr: "timed out",
		},
		{
			name:       "no route to host",
			stderr:     "ssh: connect to host 10.0.0.142 port 2222: No route to host",
			wantSubstr: "no route to host",
		},
		{
			name:       "could not resolve",
			stderr:     "ssh: Could not resolve hostname bogus: nodename nor servname provided",
			wantSubstr: "could not resolve host",
		},
		{
			name:       "permission denied",
			stderr:     "Permission denied (publickey).",
			wantSubstr: "permission denied",
		},
		{
			name:       "host key changed",
			stderr:     "@@@ WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED! @@@",
			wantSubstr: "host key",
		},
		{
			name:       "unknown stderr passthrough",
			stderr:     "something unexpected happened",
			wantSubstr: "something unexpected happened",
		},
		{
			name:       "empty stderr falls back to run error",
			stderr:     "",
			wantSubstr: "exit status 255",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifySSHError(cfg, runErr, tt.stderr)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.wantSubstr)) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}
