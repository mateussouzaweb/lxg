package run

import (
	"context"
	"fmt"

	"github.com/mateussouzaweb/lxg/command"
)

// LinkPipewire socket file
func LinkPipewire(ctx *command.Context, cancel context.Context) error {

	userPipewire := fmt.Sprintf("/run/user/%s/pipewire-0", ctx.UID)
	lxgPipewire := fmt.Sprintf("/run/user/%s/lxg/pipewire-0", ctx.UID)
	err := BindMount(userPipewire, lxgPipewire, cancel)
	if err != nil {
		return fmt.Errorf("pipewire error: %w", err)
	}

	return nil
}
