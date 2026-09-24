package systemd

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/mateussouzaweb/lxg/command"
)

//go:embed services/*.ini
var servicesFS embed.FS

// InstallListenService to automatically runs the process
func InstallListenService(ctx *command.Context) error {

	fmt.Printf("Checking LXG listen service...\n")
	serviceName := "lxg-listen.service"

	// Skip when already active
	active, err := IsServiceActive(serviceName)
	if err != nil {
		return fmt.Errorf("systemd service check error: %w", err)
	} else if active {
		fmt.Printf("LXG listen service already active.\n")
		return nil
	}

	// Install and start service
	fmt.Printf("Activating LXG listen service.\n")
	serviceSpec, err := servicesFS.ReadFile("services/listen.ini")
	if err != nil {
		return fmt.Errorf("systemd service read error: %w", err)
	}

	servicePath := ServicePath(ctx.Home)
	serviceFile := filepath.Join(servicePath, serviceName)
	err = AddService(serviceName, serviceFile, serviceSpec)
	if err != nil {
		return fmt.Errorf("systemd error: %w", err)
	}

	fmt.Printf("LXG listen service activated.\n")
	return nil
}
