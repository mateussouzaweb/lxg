package container

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mateussouzaweb/lxg/context"
)

// RunCommand on container based on given args
func RunCommand(ctx *context.Context) error {

	command := ctx.Args[0]
	args := []string{}

	if len(ctx.Args) > 1 {
		args = append(args, ctx.Args[1:]...)
	}

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("run command error: %w", err)
	}

	return nil
}
