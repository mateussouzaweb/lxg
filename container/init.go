package container

import "github.com/mateussouzaweb/lxg/command"

// Init LXG environment support
func Init(ctx *command.Context) error {

	// Ensure D-Bus router is running
	err := EnsureRouter(ctx)
	if err != nil {
		return err
	}

	// Init environment on container
	err = InitEnvironment(ctx)
	if err != nil {
		return err
	}

	return nil
}
