//go:build !windows

package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/chris-roerig/homegit/internal/config"
)

func Start(cfg *config.Config) error {
	if IsRunning(cfg) {
		return fmt.Errorf("server is already running")
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	logDir := filepath.Dir(cfg.PIDFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logFile, err := os.OpenFile(filepath.Join(logDir, "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command(exe, "serve")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := os.WriteFile(cfg.PIDFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644); err != nil {
		return err
	}

	cmd.Process.Release()
	return nil
}

func Stop(cfg *config.Config) error {
	pid, err := readPID(cfg.PIDFile)
	if err != nil {
		return fmt.Errorf("server is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(cfg.PIDFile)
		return fmt.Errorf("server is not running")
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		os.Remove(cfg.PIDFile)
		return fmt.Errorf("failed to stop server: %w", err)
	}

	os.Remove(cfg.PIDFile)
	return nil
}

func Status(cfg *config.Config) error {
	if IsRunning(cfg) {
		pid, _ := readPID(cfg.PIDFile)
		fmt.Printf("Server is running (PID: %d)\n", pid)
		return nil
	}
	fmt.Println("Server is not running")
	return nil
}

func IsRunning(cfg *config.Config) bool {
	pid, err := readPID(cfg.PIDFile)
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}
