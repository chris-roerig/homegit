package git

import (
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantType    string
		wantRepo    string
		expectError bool
	}{
		{
			name:        "upload-pack with single quotes",
			input:       "git-upload-pack 'my-repo.git'",
			wantType:    "upload-pack",
			wantRepo:    "my-repo.git",
			expectError: false,
		},
		{
			name:        "receive-pack with double quotes",
			input:       "git-receive-pack \"my-repo.git\"",
			wantType:    "receive-pack",
			wantRepo:    "my-repo.git",
			expectError: false,
		},
		{
			name:        "upload-pack without quotes",
			input:       "git-upload-pack my-repo.git",
			wantType:    "upload-pack",
			wantRepo:    "my-repo.git",
			expectError: false,
		},
		{
			name:        "invalid command",
			input:       "git-invalid my-repo.git",
			expectError: true,
		},
		{
			name:        "missing repo path",
			input:       "git-upload-pack",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ParseCommand(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cmd.Type != tt.wantType {
				t.Errorf("Expected type %s, got %s", tt.wantType, cmd.Type)
			}

			if cmd.RepoPath != tt.wantRepo {
				t.Errorf("Expected repo %s, got %s", tt.wantRepo, cmd.RepoPath)
			}
		})
	}
}
