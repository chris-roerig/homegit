package cmd

import (
	"fmt"

	"github.com/chris-roerig/homegit/internal/config"
	"github.com/chris-roerig/homegit/internal/daemon"
)

// versionLabel returns the standard prefix used for daemon-related command
// output, e.g. "homegit 1.2.0:". Centralizing it keeps the format consistent
// across start/stop/restart/status/serve.
func versionLabel() string {
	return fmt.Sprintf("homegit %s:", Version)
}

func Start(cfg *config.Config) error {
	if err := daemon.Start(cfg); err != nil {
		return err
	}
	fmt.Printf("%s server started\n", versionLabel())
	return nil
}

func Stop(cfg *config.Config) error {
	if err := daemon.Stop(cfg); err != nil {
		return err
	}
	fmt.Printf("%s server stopped\n", versionLabel())
	return nil
}

func Restart(cfg *config.Config) error {
	if daemon.IsRunning(cfg) {
		if err := daemon.Stop(cfg); err != nil {
			return err
		}
		fmt.Printf("%s server stopped\n", versionLabel())
	}
	if err := daemon.Start(cfg); err != nil {
		return err
	}
	fmt.Printf("%s server started\n", versionLabel())
	return nil
}

func Status(cfg *config.Config) error {
	if daemon.IsRunning(cfg) {
		fmt.Printf("%s server is running (PID: %d)\n", versionLabel(), daemon.PID(cfg))
		return nil
	}
	fmt.Printf("%s server is not running\n", versionLabel())
	return nil
}
