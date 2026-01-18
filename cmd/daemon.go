package cmd

import (
	"fmt"

	"github.com/chris-roerig/homegit/internal/config"
	"github.com/chris-roerig/homegit/internal/daemon"
)

func Start(cfg *config.Config) error {
	if err := daemon.Start(cfg); err != nil {
		return err
	}
	fmt.Println("Server started")
	return nil
}

func Stop(cfg *config.Config) error {
	if err := daemon.Stop(cfg); err != nil {
		return err
	}
	fmt.Println("Server stopped")
	return nil
}

func Restart(cfg *config.Config) error {
	if daemon.IsRunning(cfg) {
		if err := daemon.Stop(cfg); err != nil {
			return err
		}
		fmt.Println("Server stopped")
	}
	if err := daemon.Start(cfg); err != nil {
		return err
	}
	fmt.Println("Server started")
	return nil
}

func Status(cfg *config.Config) error {
	return daemon.Status(cfg)
}
