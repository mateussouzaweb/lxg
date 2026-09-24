package container

import (
	"fmt"

	"github.com/mateussouzaweb/lxg/command"
	"github.com/mateussouzaweb/lxg/container/setup"
)

// Setup LXG support on container (self installer)
func Setup(ctx *command.Context) error {

	fmt.Printf("Checking user requirement...\n")
	err := setup.EnsureUID(ctx)
	if err != nil {
		return fmt.Errorf("setup error: %w", err)
	}

	fmt.Printf("Installing helper scripts...\n")
	err = setup.InstallInitScript(ctx)
	if err != nil {
		return fmt.Errorf("setup error: %w", err)
	}

	fmt.Printf("Container has been configured.\n")
	return nil
}
