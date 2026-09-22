package container

import (
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

	list := map[string]string{
		hostPath("/pulse/native"): userPath("/pulse/native"),
		hostPath("/pipewire-0"):   userPath("/pipewire-0"),
		hostPath("/wayland-0"):    userPath("/wayland-0"),
		hostPath("/lxg.host.bus"): userPath("/lxg.host.bus"),
		hostPath("/lxg.sock"):     userPath("/lxg.sock"),
		"/lxg/tmp/.X11-unix/X0":   "/tmp/.X11-unix/X0",
		"/lxg/tmp/.X11-unix/X1":   "/tmp/.X11-unix/X1",
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
		if err != nil {
			return err
		}

		return nil
	}

	// Symlink host runtime sockets
	for source, destination := range list {

		// Check if symlink already is pointing to host socket
		match, err := symlinkMatch(source, destination)
		if err != nil {
			return err
		} else if match {
			continue
		}

		// Remove current entry if necessary
		err = removeExisting(destination)
		if err != nil {
			return err
		}

		// Make symlink to host socket
		err = os.Symlink(source, destination)
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

	// Set dBus address to router if present, fallback to native bus or host bus
	dBusAddress := ""
	if _, err := os.Stat(userPath("/lxg.router.bus")); err == nil {
		dBusAddress = fmt.Sprintf("unix:path=%s", userPath("/lxg.router.bus"))
	} else if _, err := os.Stat(userPath("/bus")); err == nil {
		dBusAddress = fmt.Sprintf("unix:path=%s", userPath("/bus"))
	} else if _, err := os.Stat(hostPath("/lxg.host.bus")); err == nil {
		dBusAddress = fmt.Sprintf("unix:path=%s", hostPath("/lxg.host.bus"))
	}

	// Export environment variables
	xdgDesktop := "GNOME"
	xdgMenuPrefix := "gnome-"
	xdgRuntimeDir := userPath("")
	pulseServer := fmt.Sprintf("unix:%s", userPath("/pulse/native"))
	variables := map[string]string{
		"LXG_CONTAINER":       "1",
		"DISPLAY":             ":0",
		"WAYLAND_DISPLAY":     "wayland-0",
		"XDG_SESSION_TYPE":    "wayland",
		"XDG_RUNTIME_DIR":     xdgRuntimeDir,
		"XDG_CURRENT_DESKTOP": xdgDesktop,
		"XDG_MENU_PREFIX":     xdgMenuPrefix,
		"PULSE_SERVER":        pulseServer,
	}

	if mutterXAuth != "" {
		variables["XAUTHORITY"] = mutterXAuth
	}
	if dBusAddress != "" {
		variables["DBUS_SESSION_BUS_ADDRESS"] = dBusAddress
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
