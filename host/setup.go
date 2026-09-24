package host

import (
	"fmt"

	"github.com/mateussouzaweb/lxg/command"
	"github.com/mateussouzaweb/lxg/host/lxc"
	"github.com/mateussouzaweb/lxg/host/systemd"
)

// Setup host and container
func Setup(ctx *command.Context) error {

	err := systemd.InstallListenService(ctx)
	if err != nil {
		return fmt.Errorf("setup error: %w", err)
	}

	err = lxc.Setup(ctx)
	if err != nil {
		return fmt.Errorf("setup error: %w", err)
	}

	fmt.Printf("Host and container has been configured.\n")
	return nil
}
