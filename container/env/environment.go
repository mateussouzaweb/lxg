package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mateussouzaweb/lxg/command"
)

// InitEnvironment on container to add LXG support
func InitEnvironment(ctx *command.Context) error {

	// User must be greater than 1000 to enable desktop session configuration
	uid := os.Getuid()
	if uid < 1000 {
		return nil
	}

	hostRuntime := fmt.Sprintf("/lxg/run/user/%d", uid)
	hostPath := func(path string) string {
		return fmt.Sprintf("%s%s", hostRuntime, path)
	}

	userRuntime := fmt.Sprintf("/run/user/%d", uid)
	userPath := func(path string) string {
		return fmt.Sprintf("%s%s", userRuntime, path)
	}

	// Ensure runtime directory exists
	err := os.MkdirAll(userPath(""), 0700)
	if err != nil {
		return err
	}

	err = os.MkdirAll(userPath("/pulse"), 0700)
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
		hostPath("/pulse/native"): userPath("/pulse/native"),
		hostPath("/pipewire-0"):   userPath("/pipewire-0"),
		hostPath("/wayland-0"):    userPath("/wayland-0"),
		hostPath("/lxg.bus"):      userPath("/lxg.bus"),
		hostPath("/lxg.bridge"):   userPath("/lxg.bridge"),
		"/lxg/tmp/.X11-unix/X0":   "/tmp/.X11-unix/X0",
		"/lxg/tmp/.X11-unix/X1":   "/tmp/.X11-unix/X1",
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
	pulseServer := fmt.Sprintf("unix:%s", userPath("/pulse/native"))
	variables := map[string]string{
		"LXG_CONTAINER":   "1",
		"DISPLAY":         ":0",
		"WAYLAND_DISPLAY": "wayland-0",
		"PULSE_SERVER":    pulseServer,
	}

	// Find Mutter XWayland authentication file
	mutterXAuthGlobPattern := filepath.Join(hostRuntime, ".mutter-Xwaylandauth.*")
	mutterXAuthMatches, err := filepath.Glob(mutterXAuthGlobPattern)
	if err != nil {
		return err
	} else if len(mutterXAuthMatches) > 0 {
		mutterXAuth := mutterXAuthMatches[0]
		variables["XAUTHORITY"] = mutterXAuth
	}

	// Set D-Bus address to first available address
	// Priority: router, proxy, native bus
	dBusAddress := ""
	dBusSources := []string{
		userPath("/lxg.router"),
		userPath("/lxg.bus"),
		userPath("/bus"),
	}

	for _, dBusPath := range dBusSources {
		_, err := os.Stat(dBusPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		} else if err != nil && errors.Is(err, os.ErrNotExist) {
			continue
		}

		dBusAddress = fmt.Sprintf("unix:path=%s", dBusPath)
		break
	}

	if dBusAddress != "" {
		variables["DBUS_SESSION_BUS_ADDRESS"] = dBusAddress
	}

	// Append XDG details
	xdgDesktop := "GNOME"
	xdgMenuPrefix := "gnome-"
	xdgRuntimeDir := userPath("")

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
