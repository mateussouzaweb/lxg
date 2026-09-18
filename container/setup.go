package container

import (
	"fmt"

	"github.com/mateussouzaweb/lxg/context"
)

// Setup LXG support on container (self installer)
func Setup(ctx *context.Context) error {

	err := EnsureUID(ctx)
	if err != nil {
		return fmt.Errorf("setup error: %w", err)
	}

	err = InstallScripts(ctx)
	if err != nil {
		return fmt.Errorf("setup error: %w", err)
	}

	err = InstallDependencies(ctx)
	if err != nil {
		return fmt.Errorf("setup error: %w", err)
	}

	fmt.Printf("Container has been configured.\n")
	return nil
}
