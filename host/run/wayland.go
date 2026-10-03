package run

import (
	"context"
	"fmt"

	"github.com/mateussouzaweb/lxg/command"
)

// LinkWayland socket file
func LinkWayland(ctx *command.Context, cancel context.Context) error {

	userWayland := fmt.Sprintf("/run/user/%s/wayland-0", ctx.UID)
	lxgWayland := fmt.Sprintf("/run/user/%s/lxg/wayland-0", ctx.UID)
	err := BindMount(userWayland, lxgWayland, cancel)
	if err != nil {
		return fmt.Errorf("wayland error: %w", err)
	}

	return nil
}
