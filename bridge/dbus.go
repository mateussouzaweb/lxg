package bridge

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mateussouzaweb/lxg/command"
)

// Init DBUS proxy listener to receive communication
func InitDBus(ctx *command.Context, cancel context.Context) error {

	// Path to LXG DBUS proxy
	proxyPath := fmt.Sprintf("/run/user/%s/lxg.bus", ctx.UID)

	// Only run if detect DBUS session address
	sessionAddress := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if sessionAddress == "" {
		fmt.Printf("WARNING!\n")
		fmt.Printf("Looks like you are not running any DBUS session..\n")
		fmt.Printf("Please make sure to run a interactive DBUS session first.\n")
		return nil
	}

	// Only run if package is installed
	_, err := exec.LookPath("xdg-dbus-proxy")
	if err != nil {
		fmt.Printf("WARNING!\n")
		fmt.Printf("No xdg-dbus-proxy command detected.\n")
		fmt.Printf("Please make sure to install it on host.\n")
		return nil
	}

	// Remove old proxy if exists
	err = os.Remove(proxyPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove old dbus proxy error: %w", err)
	}

	// Ensure directory exists
	err = os.MkdirAll(filepath.Dir(proxyPath), 0700)
	if err != nil {
		return fmt.Errorf("create dbus proxy directory error: %w", err)
	}

	defer os.Remove(proxyPath)

	// Start DBUS proxy
	args := []string{
		sessionAddress,
		proxyPath,
		"--filter",
		"--talk=org.freedesktop.portal.*",
		"--talk=org.freedesktop.secrets",
		"--talk=org.freedesktop.Notifications",
		"--talk=org.freedesktop.ScreenSaver",
		"--talk=org.kde.StatusNotifierWatcher",
		"--own=org.mpris.MediaPlayer2.*",
		"--log",
	}

	cmd := exec.CommandContext(cancel, "xdg-dbus-proxy", args...)
	cmd.WaitDelay = 2 * time.Second
	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("dbus proxy error: %w", err)
	}

	fmt.Printf("Bridge DBus proxy listening on %s\n", proxyPath)

	err = cmd.Wait()
	if err != nil && cancel.Err() == nil {
		return fmt.Errorf("dbus proxy stopped with error: %w", err)
	}

	return nil
}
