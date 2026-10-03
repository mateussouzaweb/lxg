package container

import (
	"errors"
	"fmt"
	"os"

	"github.com/mateussouzaweb/lxg/command"
	"github.com/mateussouzaweb/lxg/container/dbus"
	"github.com/mateussouzaweb/lxg/container/env"
)

// SymLinkRun folder to match host format
func SymLinkRun(ctx *command.Context) error {

	source := fmt.Sprintf("/lxg/run/user/%s", ctx.UID)
	destination := fmt.Sprintf("/run/user/%s/lxg", ctx.UID)

	// Remove current entry if necessary
	err := os.Remove(destination)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("symlink error: %w", err)
	}

	// Make symlink to source
	err = os.Symlink(source, destination)
	if err != nil {
		return fmt.Errorf("symlink error: %w", err)
	}

	return nil
}

// Init LXG environment support
func Init(ctx *command.Context) error {

	// Symlink run folder
	err := SymLinkRun(ctx)
	if err != nil {
		return err
	}

	// Ensure D-Bus router is running
	err = dbus.EnsureRouter(ctx)
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
