package lxc

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mateussouzaweb/lxg/command"
)

// RunCommand on container based on given args
func RunCommand(ctx *command.Context) error {

	// Make sure container is running
	err := StartContainer(ctx)
	if err != nil {
		return fmt.Errorf("container error: %w", err)
	}

	// Call LXC exec on container
	// This pattern trigger the interactive session on user
	args := append([]string{
		"exec", ctx.Container, "--",
		"sudo", "-i", "-u", fmt.Sprintf("#%s", ctx.UID),
	}, ctx.Args...)

	cmd := exec.Command("lxc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("run command error: %w", err)
	}

	return nil
}
