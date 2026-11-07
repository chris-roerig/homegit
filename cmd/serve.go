package cmd

import (
	"fmt"

	"github.com/chris-roerig/homegit/internal/config"
	"github.com/chris-roerig/homegit/internal/ssh"
)

func Serve(cfg *config.Config) error {
	fmt.Printf("%s server listening on port %d\n", versionLabel(), cfg.Server.Port)
	server, err := ssh.NewServer(cfg)
	if err != nil {
		return err
	}
	return server.Start()
}
