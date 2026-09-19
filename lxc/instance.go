package lxc

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/mateussouzaweb/lxg/command"
)

// ContainerStatus return the status of the container
func ContainerStatus(ctx *command.Context) (string, error) {

	// Check for status description
	args := []string{"info", ctx.Container}
	cmd := exec.Command("lxc", args...)
	cmd.Stdin = os.Stdin

	result, err := command.Execute(cmd)
	notFoundText := "Instance not found"

	if err != nil {
		return "", fmt.Errorf("status error: %w", err)
	} else if strings.Contains(result.Error, notFoundText) {
		return "", fmt.Errorf("status error: container %s not found", ctx.Container)
	} else if result.ExitCode >= 1 {
		return "", fmt.Errorf("status error: %s", result.Error)
	}

	// Extract status value
	statusRegex := regexp.MustCompile(`Status:\s+([a-zA-Z]+)`)
	matches := statusRegex.FindStringSubmatch(result.Output)

	if len(matches) > 1 {
		statusText := strings.ToUpper(matches[1])
		return statusText, nil
	}

	return "UNKNOWN", nil
}

// StartContainer if not started yet
func StartContainer(ctx *command.Context) error {

	// Check if already is running
	status, err := ContainerStatus(ctx)
	if err != nil {
		return fmt.Errorf("start error: %w", err)
	} else if status == "RUNNING" {
		return nil
	}

	// Start container when not running
	args := []string{"start", ctx.Container}
	cmd := exec.Command("lxc", args...)
	cmd.Stdin = os.Stdin

	result, err := command.Execute(cmd)
	if err != nil {
		return fmt.Errorf("start error: %w", err)
	} else if result.ExitCode >= 1 {
		return fmt.Errorf("start error: %s", result.Error)
	}

	// Wait and try again
	time.Sleep(1 * time.Second)
	return StartContainer(ctx)
}

// StopContainer if not stopped yet
func StopContainer(ctx *command.Context) error {

	// Check if already is stopped
	status, err := ContainerStatus(ctx)
	if err != nil {
		return fmt.Errorf("stop error: %w", err)
	} else if status == "STOPPED" {
		return nil
	}

	// Stop container when running
	args := []string{"stop", ctx.Container}
	cmd := exec.Command("lxc", args...)
	cmd.Stdin = os.Stdin

	result, err := command.Execute(cmd)
	if err != nil {
		return fmt.Errorf("stop error: %w", err)
	} else if result.ExitCode >= 1 {
		return fmt.Errorf("stop error: %s", result.Error)
	}

	// Wait and try again
	time.Sleep(1 * time.Second)
	return StopContainer(ctx)
}

// RestartContainer by stop and starting again
func RestartContainer(ctx *command.Context) error {

	err := StopContainer(ctx)
	if err != nil {
		return fmt.Errorf("restart error: %w", err)
	}

	err = StartContainer(ctx)
	if err != nil {
		return fmt.Errorf("restart error: %w", err)
	}

	return nil
}
