package cmd

import (
	"context"
	"fmt"
	"net"

	"github.com/chris-roerig/homegit/internal/config"
	"github.com/chris-roerig/homegit/internal/netacl"
	"github.com/chris-roerig/homegit/internal/ssh"
	"github.com/chris-roerig/homegit/internal/web"
)

// Serve runs the SSH server and, unless disabled by noWeb or the
// server.web_enabled config setting, the read-only web repo browser.
func Serve(cfg *config.Config, noWeb bool) error {
	fmt.Printf("%s server listening on port %d\n", versionLabel(), cfg.Server.Port)
	server, err := ssh.NewServer(cfg) // also validates server.allowed_ips
	if err != nil {
		return err
	}
	if n := len(cfg.Server.AllowedIPs); n > 0 {
		fmt.Printf("Access restricted to %d allowed_ips entries (loopback always allowed)\n", n)
	}

	if !noWeb && cfg.Server.WebEnabled {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		startWeb(ctx, cfg)
	}

	return server.Start()
}

// startWeb launches the web browser in the background. A failure (for example
// the port being in use) is logged and does not stop the SSH server.
func startWeb(ctx context.Context, cfg *config.Config) {
	srv, err := web.New(cfg.Server.ReposDir, cfg.Server.Port)
	if err != nil {
		fmt.Printf("Web UI disabled: %v\n", err)
		return
	}
	addr := fmt.Sprintf(":%d", cfg.Server.WebPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("Web UI disabled: %v\n", err)
		return
	}
	acl, err := netacl.Parse(cfg.Server.AllowedIPs)
	if err != nil {
		fmt.Printf("Web UI disabled: %v\n", err)
		_ = ln.Close()
		return
	}
	ln = netacl.Wrap(ln, acl, "Web UI", func(f string, a ...any) { fmt.Printf(f+"\n", a...) })
	fmt.Printf("Web UI listening on port %d\n", cfg.Server.WebPort)
	go func() {
		if err := srv.Serve(ctx, ln); err != nil {
			fmt.Printf("Web UI stopped: %v\n", err)
		}
	}()
}
