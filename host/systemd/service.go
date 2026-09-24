package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mateussouzaweb/lxg/command"
)

// ServicePath return the systemd path for user services
func ServicePath(home string) string {
	return filepath.Join(home, ".config", "systemd", "user")
}

// IsServiceActive checks if the service is currently running
func IsServiceActive(serviceName string) (bool, error) {

	args := []string{"--user", "is-active", "--quiet", serviceName}
	cmd := exec.Command("systemctl", args...)
	cmd.Stdin = os.Stdin

	// Returns 0 exit code if active/running
	result, err := command.Execute(cmd)
	if err != nil {
		return false, fmt.Errorf("systemd service check error: %w", err)
	} else if result.ExitCode == 0 {
		return true, nil
	}

	return false, nil
}

// EnableService on systemd
func EnableService(serviceName string) error {

	// Reload daemon just to make sure service is available
	args := []string{"--user", "daemon-reload"}
	cmd := exec.Command("systemctl", args...)
	cmd.Stdin = os.Stdin

	result, err := command.Execute(cmd)
	if err != nil {
		return fmt.Errorf("systemd daemon reload error: %w", err)
	} else if result.ExitCode > 1 {
		return fmt.Errorf("systemd daemon reload error: %s", result.Error)
	}

	// Enable the service
	args = []string{"--user", "enable", "--now", serviceName}
	cmd = exec.Command("systemctl", args...)
	cmd.Stdin = os.Stdin

	result, err = command.Execute(cmd)
	if err != nil {
		return fmt.Errorf("systemd enable service error: %w", err)
	} else if result.ExitCode > 1 {
		return fmt.Errorf("systemd enable service error: %s", result.Error)
	}

	return err
}

// AddService on systemd handler
func AddService(serviceName string, path string, specs []byte) error {

	// Ensure services directory exists
	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		return fmt.Errorf("systemd create directory error: %w", err)
	}

	// Write the service
	err = os.WriteFile(path, specs, 0755)
	if err != nil {
		return fmt.Errorf("systemd service write error: %w", err)
	}

	// Enable the service
	return EnableService(serviceName)
}
