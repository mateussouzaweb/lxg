package lxc

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/mateussouzaweb/lxg/command"
)

// Setup LXG support on host and container
func Setup(ctx *command.Context) error {

	// Setup profile on host
	fmt.Printf("Configuring profile...\n")

	created, err := SetupProfile(ctx)
	if err != nil {
		return fmt.Errorf("profile setup error: %w", err)
	} else if created {
		fmt.Printf("Profile created.\n")
	}

	// Attach profile to container
	// When attached, stop container to reflect changes
	fmt.Printf("Attaching profile to container: %s -> %s...\n", ctx.Profile, ctx.Container)

	attached, err := AttachProfile(ctx)
	if err != nil {
		return fmt.Errorf("profile attach error: %w", err)
	} else if attached {
		fmt.Printf("Profile attached to container.\n")
		fmt.Printf("Rebooting container to reflect...\n")
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
		"--from-host", fmt.Sprintf("--%s", ctx.Profile),
	}

	cmd := exec.Command("lxc", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("container error: %w", err)
	}

	// Restart container
	fmt.Printf("Restarting container...\n")

	err = RestartContainer(ctx)
	if err != nil {
		return fmt.Errorf("container error: %w", err)
	}

	return nil
}
