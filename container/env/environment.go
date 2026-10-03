package env

import (
	"errors"
	"fmt"
	"os"

	"github.com/mateussouzaweb/lxg/command"
)

// InitEnvironment on container to add LXG support
func InitEnvironment(ctx *command.Context) error {

	// User must be greater than 1000 to enable desktop session configuration
	uid := os.Getuid()
	if uid < 1000 {
		return nil
	}

	// Return user runtime path
	runPath := func(path string) string {
		return fmt.Sprintf("/run/user/%d/%s", uid, path)
	}

	// Ensure runtime directory exists
	err := os.MkdirAll(runPath("pulse"), 0700)
	if err != nil {
		return err
	}

	err = os.MkdirAll("/tmp/.X11-unix", 01777)
	if err != nil {
		return err
	}

	// Check if is symlink and match with desired source
	symlinkMatch := func(source string, destination string) (bool, error) {

		info, err := os.Lstat(destination)
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}

		if info.Mode()&os.ModeSymlink == 0 {
			return false, nil
		}

		targetLink, err := os.Readlink(destination)
		if err != nil {
			return false, err
		}

		return targetLink == source, nil
	}

	// Remove existing file on path
	removeExisting := func(path string) error {

		// Check for file presence
		_, err = os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		// Remove current entry if necessary
		err = os.Remove(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}

		return nil
	}

	// List of possible symlink points
	// Final symlink list vary based on container type
	// When host path not available, symlink is skipped
	list := map[string]string{
		runPath("lxg/pulse-native"): runPath("pulse/native"),
		runPath("lxg/pipewire-0"):   runPath("pipewire-0"),
		runPath("lxg/wayland-0"):    runPath("wayland-0"),
		runPath("lxg/X11-X0"):       "/tmp/.X11-unix/X0",
		runPath("lxg/X11-X1"):       "/tmp/.X11-unix/X1",
	}

	// Symlink host runtime sockets
	for source, destination := range list {

		// Skip when source file does not exist
		_, err := os.Stat(source)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		} else if err != nil && errors.Is(err, os.ErrNotExist) {
			continue
		}

		// Check if destination symlink already is pointing to source
		match, err := symlinkMatch(source, destination)
		if err != nil {
			return err
		} else if match {
			continue
		}

		// Remove current destination entry if necessary
		err = removeExisting(destination)
		if err != nil {
			return err
		}

		// Make symlink to source
		err = os.Symlink(source, destination)
		if err != nil {
			return err
		}

	}

	// Define environment variables
	variables := map[string]string{
		"DISPLAY":         ":0",
		"WAYLAND_DISPLAY": "wayland-0",
	}

	// Set X11 auth
	xAuthPath := runPath("lxg/X11-Xauth")
	_, err = os.Stat(xAuthPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	} else if err == nil {
		variables["XAUTHORITY"] = xAuthPath
	}

	// Set pulse server variable
	pulsePath := runPath("pulse/native")
	_, err = os.Stat(pulsePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	} else if err == nil {
		pulseServer := fmt.Sprintf("unix:%s", pulsePath)
		variables["PULSE_SERVER"] = pulseServer
	}

	// Set D-Bus address to first available address
	// Priority: router, proxy, native bus
	dBusSources := []string{
		runPath("lxg/router"),
		runPath("lxg/bus"),
		runPath("bus"),
	}

	for _, dBusPath := range dBusSources {
		_, err := os.Stat(dBusPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		} else if err != nil && errors.Is(err, os.ErrNotExist) {
			continue
		}

		dBusAddress := fmt.Sprintf("unix:path=%s", dBusPath)
		variables["DBUS_SESSION_BUS_ADDRESS"] = dBusAddress
		break
	}

	// Append XDG details
	xdgDesktop := "GNOME"
	xdgMenuPrefix := "gnome-"
	xdgRuntimeDir := runPath("")

	variables["XDG_SESSION_TYPE"] = "wayland"
	variables["XDG_RUNTIME_DIR"] = xdgRuntimeDir
	variables["XDG_CURRENT_DESKTOP"] = xdgDesktop
	variables["XDG_MENU_PREFIX"] = xdgMenuPrefix

	// Export environment variables
	// Use export statements for the shell to evaluate
	for key, value := range variables {
		fmt.Printf("export %s=%q\n", key, value)
	}

	return nil
}
