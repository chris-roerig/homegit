package cmd

import (
	"github.com/chris-roerig/homegit/internal/config"
	"github.com/chris-roerig/homegit/internal/ssh"
)

func Serve(cfg *config.Config) error {
	server, err := ssh.NewServer(cfg)
	if err != nil {
		return err
	}
	return server.Start()
}
