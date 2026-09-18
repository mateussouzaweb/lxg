package container

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mateussouzaweb/lxg/context"
)

// InitEnvironment on container to add LXG support
func InitEnvironment(ctx *context.Context) error {

	// Only main user need desktop session configuration
	uid := os.Getuid()
	if uid != allowedUID {
		return nil
	}

	runtimeDir := fmt.Sprintf("/run/user/%d", uid)
	hostRuntime := fmt.Sprintf("/lxg/run/user/%d", uid)

	// Ensure runtime directory exists
	err := os.MkdirAll(runtimeDir, 0700)
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
		if err != nil {
			return err
		}

		return nil
	}

	// Symlink host runtime sockets
	sockets := []string{"pipewire-0", "wayland-0", "bus", "lxg.sock"}
	for _, socket := range sockets {
		hostSocket := filepath.Join(hostRuntime, socket)
		targetSocket := filepath.Join(runtimeDir, socket)

		// Check if symlink already is pointing to host socket
		match, err := symlinkMatch(hostSocket, targetSocket)
		if err != nil {
			return err
		} else if match {
			continue
		}

		// Remove current entry if necessary
		err = removeExisting(targetSocket)
		if err != nil {
			return err
		}

		// Make symlink to host socket
		err = os.Symlink(hostSocket, targetSocket)
		if err != nil {
			return err
		}
	}

	// Symlink X11 display sockets
	err = os.MkdirAll("/tmp/.X11-unix", 01777)
	if err != nil {
		return err
	}

	for _, display := range []string{"X0", "X1"} {
		hostDisplay := filepath.Join("/lxg/tmp/.X11-unix", display)
		targetDisplay := filepath.Join("/tmp/.X11-unix", display)

		// Check if symlink already is pointing to host display
		match, err := symlinkMatch(hostDisplay, targetDisplay)
		if err != nil {
			return err
		} else if match {
			continue
		}

		// Remove current entry if necessary
		err = removeExisting(targetDisplay)
		if err != nil {
			return err
		}

		// Make symlink to host display
		err = os.Symlink(hostDisplay, targetDisplay)
		if err != nil {
			return err
		}
	}

	// Find Mutter XWayland authentication file
	globPattern := filepath.Join(hostRuntime, ".mutter-Xwaylandauth.*")
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return err
	}

	mutterXAuth := ""
	if len(matches) > 0 {
		mutterXAuth = matches[0]
	}

	// Export environment variables
	variables := map[string]string{
		"LXG_CONTAINER":            "1",
		"DISPLAY":                  ":0",
		"WAYLAND_DISPLAY":          "wayland-0",
		"XDG_RUNTIME_DIR":          runtimeDir,
		"PULSE_SERVER":             fmt.Sprintf("unix:%s/pipewire-0", runtimeDir),
		"DBUS_SESSION_BUS_ADDRESS": fmt.Sprintf("unix:path=%s/bus", runtimeDir),
		"XDG_CURRENT_DESKTOP":      "GNOME",
		"XDG_MENU_PREFIX":          "gnome-",
		"XAUTHORITY":               mutterXAuth,
	}

	// Use export statements for the shell to evaluate
	for key, value := range variables {
		fmt.Printf("export %s=%q\n", key, value)
	}

	return nil
}

// IsContainer checks if terminal context is from container or not
func IsContainer() (bool, error) {

	// Container has the LXG_CONTAINER environment variable
	if os.Getenv("LXG_CONTAINER") == "1" {
		return true, nil
	}

	// Container mounts host /run folder on /lxg/run
	_, err := os.Stat("/lxg/run")
	if err != nil && !os.IsNotExist(err) {
		return false, err
	} else if err != nil && os.IsNotExist(err) {
		return false, nil
	} else if err == nil {
		return true, nil
	}

	return false, nil
}
