package run

import (
	"context"
	"fmt"

	"github.com/mateussouzaweb/lxg/command"
)

// LinkPulseNative socket file
func LinkPulseNative(ctx *command.Context, cancel context.Context) error {

	userPulseNative := fmt.Sprintf("/run/user/%s/pulse/native", ctx.UID)
	lxgPulseNative := fmt.Sprintf("/run/user/%s/lxg/pulse-native", ctx.UID)
	err := BindMount(userPulseNative, lxgPulseNative, cancel)
	if err != nil {
		return fmt.Errorf("pulse native error: %w", err)
	}

	return nil
}
