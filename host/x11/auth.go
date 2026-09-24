package x11

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mateussouzaweb/lxg/command"
)

// LinkAuth for X11 authentication
func LinkAuth(ctx *command.Context, cancel context.Context) error {

	userRuntime := fmt.Sprintf("/run/user/%s", ctx.UID)
	authLink := filepath.Join(userRuntime, "lxg.xauth")

	// Find Mutter XWayland authentication file
	globPattern := filepath.Join(userRuntime, ".mutter-Xwaylandauth.*")
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return fmt.Errorf("find x11 auth error: %w", err)
	} else if len(matches) == 0 {
		return nil
	}

	xAuthFile := matches[0]

	// Remove old auth link if exists
	err = os.Remove(authLink)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove x11 auth error: %w", err)
	}

	// Make content copy to destination
	// Due this copy process, must run on every auth reload
	// Since service runs after session login, we do not expect this reload is necessary
	content, err := os.ReadFile(xAuthFile)
	if err != nil {
		return fmt.Errorf("read x11 auth error: %w", err)
	}

	err = os.WriteFile(authLink, content, 0700)
	if err != nil {
		return fmt.Errorf("link x11 auth error: %w", err)
	}

	// Remove auth link when context is cancelled
	go func() {
		<-cancel.Done()
		os.Remove(authLink)
	}()

	return nil
}
