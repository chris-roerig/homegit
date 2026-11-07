//go:build windows

package daemon

import (
	"fmt"

	"github.com/chris-roerig/homegit/internal/config"
)

func Start(cfg *config.Config) error {
	return fmt.Errorf("daemon mode is not supported on Windows. Use 'homegit serve' instead")
}

func Stop(cfg *config.Config) error {
	return fmt.Errorf("daemon mode is not supported on Windows")
}

func Status(cfg *config.Config) error {
	fmt.Println("Daemon mode is not supported on Windows")
	fmt.Println("Use 'homegit serve' to run in foreground")
	return nil
}

func IsRunning(cfg *config.Config) bool {
	return false
}

// PID returns 0 on Windows, where daemon mode is unsupported.
func PID(cfg *config.Config) int {
	return 0
}
