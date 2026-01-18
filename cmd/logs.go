package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/chris-roerig/homegit/internal/config"
)

func Logs(cfg *config.Config, args []string) error {
	logFile := filepath.Join(filepath.Dir(cfg.PIDFile), "server.log")

	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return fmt.Errorf("no log file found at %s", logFile)
	}

	lines := 50
	follow := false

	for i, arg := range args {
		if arg == "--tail" || arg == "-n" {
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil {
					lines = n
				}
			}
		}
		if arg == "--follow" || arg == "-f" {
			follow = true
		}
	}

	var cmd *exec.Cmd
	if follow {
		cmd = exec.Command("tail", "-f", "-n", strconv.Itoa(lines), logFile)
	} else {
		cmd = exec.Command("tail", "-n", strconv.Itoa(lines), logFile)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}
