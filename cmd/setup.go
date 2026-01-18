package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/chris-roerig/homegit/internal/config"
)

func Setup() error {
	fmt.Println("homegit setup")
	fmt.Println("=============")
	fmt.Println()

	fmt.Println("Where should your Git repositories be stored?")
	fmt.Println()
	fmt.Println("  1. On this computer (most common)")
	fmt.Println("     - Use homegit on this computer")
	fmt.Println("     - Repos are stored locally")
	fmt.Println()
	fmt.Println("  2. On another computer")
	fmt.Println("     - You already set up homegit on another computer")
	fmt.Println("     - This computer will connect to it")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Ask if this is the server
	fmt.Print("Store repos on this computer? (Y/n): ")
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))

	cfg := config.Default()

	if response == "" || response == "y" || response == "yes" {
		// This is the server
		hostname, _ := os.Hostname()
		fmt.Printf("\n✓ Repos will be stored on this computer\n")
		fmt.Printf("✓ Other computers can connect using: %s\n", hostname)
		cfg.ServerHost = "localhost"
	} else {
		// This is a client
		fmt.Print("\nEnter the computer's hostname or IP where repos are stored: ")
		serverHost, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		serverHost = strings.TrimSpace(serverHost)

		if serverHost == "" {
			return fmt.Errorf("hostname/IP is required")
		}

		// Basic validation - no spaces, not localhost variants
		if strings.Contains(serverHost, " ") {
			return fmt.Errorf("invalid hostname/IP: cannot contain spaces")
		}
		if serverHost == "localhost" || serverHost == "127.0.0.1" {
			return fmt.Errorf("use option 1 to store repos on this computer")
		}

		cfg.ServerHost = serverHost
		fmt.Printf("\n✓ Will connect to: %s\n", serverHost)
	}

	// Save config
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println("\n✓ Configuration saved to ~/.homegit/config")

	if cfg.ServerHost == "localhost" {
		// This is the server - ask if they want to start it
		fmt.Print("\nStart homegit now? (Y/n): ")
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "" || response == "y" || response == "yes" {
			fmt.Println()
			if err := Start(cfg); err != nil {
				fmt.Printf("⚠ Failed to start: %v\n", err)
				fmt.Println("\nYou can start it manually with: homegit start")
			}
		} else {
			fmt.Println("\nNext steps:")
			fmt.Println("  1. Start homegit:     homegit start")
			fmt.Println("  2. Initialize a repo: homegit init")
		}
	} else {
		fmt.Println("\nNext steps:")
		fmt.Printf("  1. Make sure homegit is running on %s\n", cfg.ServerHost)
		fmt.Println("  2. Initialize a repo: homegit init")
	}

	return nil
}
