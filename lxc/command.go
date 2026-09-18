package lxc

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mateussouzaweb/lxg/context"
)

// RunCommand on container based on given args
func RunCommand(ctx *context.Context) error {

	// Make sure container is running
	err := StartContainer(ctx)
	if err != nil {
		return fmt.Errorf("container error: %w", err)
	}

	// Call LXC exec on container
	args := append([]string{
		"exec", ctx.Container, "--",
		"su", "--login", ctx.UID,
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
