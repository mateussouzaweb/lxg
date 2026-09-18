package lxc

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mateussouzaweb/lxg/command"
	"github.com/mateussouzaweb/lxg/context"
)

// Setup LXG support on host and container
func Setup(ctx *context.Context) error {

	// Setup profile on host
	fmt.Printf("Configuring desktop profile...\n")

	created, err := SetupProfile(ctx)
	if err != nil {
		return fmt.Errorf("profile setup error: %w", err)
	} else if created {
		fmt.Printf("Desktop profile created.\n")
	}

	// Attach profile to container
	// When attached, stop container to reflect changes
	attached, err := AttachProfile(ctx)
	if err != nil {
		return fmt.Errorf("profile attach error: %w", err)
	} else if attached {
		fmt.Printf("Desktop profile attached to container.\n")
		fmt.Printf("Rebooting container...\n")
		err = StopContainer(ctx)
		if err != nil {
			return fmt.Errorf("profile attach error: %w", err)
		}
	}

	// Make sure container is started
	err = StartContainer(ctx)
	if err != nil {
		return fmt.Errorf("container error: %w", err)
	}

	// Run command to install scripts inside container
	// LXG should now be available inside it
	fmt.Printf("Executing container internal setup...\n")

	args := []string{
		"exec", ctx.Container, "--",
		"/usr/local/bin/lxg", "container", "setup",
	}

	cmd := exec.Command("lxc", args...)
	cmd.Stdin = os.Stdin

	result, err := command.Execute(cmd)
	if err != nil {
		return fmt.Errorf("container error: %w", err)
	} else if result.ExitCode >= 1 {
		return fmt.Errorf("container error: %s", result.Error)
	}

	// Restart container
	fmt.Printf("Restarting container...\n")

	err = RestartContainer(ctx)
	if err != nil {
		return fmt.Errorf("container error: %w", err)
	}

	fmt.Printf("Host and container has been configured.\n")
	return nil
}
