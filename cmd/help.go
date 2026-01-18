package cmd

import (
	"fmt"
	"net"
	"os"

	"github.com/chris-roerig/homegit/internal/config"
)

func Help(cfg *config.Config) error {
	hostname, _ := os.Hostname()
	ip := getLocalIP()

	fmt.Println("homegit - minimal portable Git server")
	fmt.Println("\n=== COMMANDS ===")
	fmt.Println("  setup       Configure homegit (server or client)")
	fmt.Println("  init        Initialize git repo and add remote")
	fmt.Println("  config      Edit configuration")
	fmt.Println("  serve       Start server in foreground")
	fmt.Println("  start       Start server as daemon")
	fmt.Println("  stop        Stop daemon server")
	fmt.Println("  restart     Restart daemon server")
	fmt.Println("  status      Check daemon status")
	fmt.Println("  list        List all repositories")
	fmt.Println("  clone       Clone a repository from the server")
	fmt.Println("  backup      Backup a repository to tar.gz")
	fmt.Println("  remove      Remove a repository")
	fmt.Println("  logs        View server logs")
	fmt.Println("  version     Show version")
	fmt.Println("  help        Show this help message")

	fmt.Println("\n=== SERVER INFO ===")
	fmt.Printf("  Port:        %d\n", cfg.Port)
	fmt.Printf("  Repos Dir:   %s\n", cfg.ReposDir)
	fmt.Printf("  Hostname:    %s\n", hostname)
	fmt.Printf("  Local IP:    %s\n", ip)

	fmt.Println("\n=== USAGE EXAMPLES ===")
	fmt.Println("\n1. Create and push a new repository:")
	fmt.Println("   cd my-project")
	fmt.Println("   git init")
	fmt.Println("   git add .")
	fmt.Println("   git commit -m \"Initial commit\"")
	fmt.Printf("   git remote add origin ssh://localhost:%d/my-project.git\n", cfg.Port)
	fmt.Println("   git push -u origin main")

	fmt.Println("\n2. Clone from this computer:")
	fmt.Printf("   git clone ssh://localhost:%d/my-project.git\n", cfg.Port)
	fmt.Println("   # Or use the shortcut:")
	fmt.Println("   homegit clone my-project")

	fmt.Println("\n3. Clone from another computer on your network:")
	fmt.Printf("   git clone ssh://%s:%d/my-project.git\n", ip, cfg.Port)

	fmt.Println("\n4. Add remote from another computer:")
	fmt.Println("   cd existing-project")
	fmt.Printf("   git remote add origin ssh://%s:%d/my-project.git\n", ip, cfg.Port)
	fmt.Println("   git push -u origin main")

	fmt.Println("\n=== NOTES ===")
	fmt.Println("  • Repositories are auto-created on first push")
	fmt.Println("  • Repository names must end with .git")
	fmt.Println("  • No authentication - use on trusted networks only")
	fmt.Printf("  • Server logs: ~/.homegit/server.log\n")

	return nil
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "unknown"
}
