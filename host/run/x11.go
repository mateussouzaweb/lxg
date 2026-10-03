package run

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mateussouzaweb/lxg/command"
)

// LinkX11Zero socket file
func LinkX11Zero(ctx *command.Context, cancel context.Context) error {

	userX11Zero := "/tmp/.X11-unix/X0"
	lxgX11Zero := fmt.Sprintf("/run/user/%s/lxg/X11-X0", ctx.UID)
	err := BindMount(userX11Zero, lxgX11Zero, cancel)
	if err != nil {
		return fmt.Errorf("x11 zero error: %w", err)
	}

	return nil
}

// LinkX11One socket file
func LinkX11One(ctx *command.Context, cancel context.Context) error {

	userX11One := "/tmp/.X11-unix/X1"
	lxgX11One := fmt.Sprintf("/run/user/%s/lxg/X11-X1", ctx.UID)
	err := BindMount(userX11One, lxgX11One, cancel)
	if err != nil {
		return fmt.Errorf("x11 one error: %w", err)
	}

	return nil
}

// LinkX11Auth authentication file
func LinkX11Auth(ctx *command.Context, cancel context.Context) error {

	lxgX11Auth := fmt.Sprintf("/run/user/%s/lxg/X11-Xauth", ctx.UID)

	// Find Mutter XWayland authentication file
	userRuntime := fmt.Sprintf("/run/user/%s", ctx.UID)
	globPattern := filepath.Join(userRuntime, ".mutter-Xwaylandauth.*")
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return fmt.Errorf("find x11 auth error: %w", err)
	}

	userX11Auth := matches[0]
	err = BindMount(userX11Auth, lxgX11Auth, cancel)
	if err != nil {
		return fmt.Errorf("x11 auth error: %w", err)
	}

	return nil
}
