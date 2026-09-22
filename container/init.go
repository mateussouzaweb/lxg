package container

import (
	"github.com/mateussouzaweb/lxg/command"
	"github.com/mateussouzaweb/lxg/container/dbus"
	"github.com/mateussouzaweb/lxg/container/env"
)

// Init LXG environment support
func Init(ctx *command.Context) error {

	// Ensure D-Bus router is running
	err := dbus.EnsureRouter(ctx)
	if err != nil {
		return err
	}

	// Init environment on container
	err = env.InitEnvironment(ctx)
	if err != nil {
		return err
	}

	return nil
}
