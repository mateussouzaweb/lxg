package dbus

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/mateussouzaweb/lxg/command"
)

// EnsureRouter checks if the D-Bus router is running and starts it if necessary
func EnsureRouter(ctx *command.Context) error {

	// Isolated containers does not have the host proxy socket
	// Do not run D-Bus router if host proxy is not present
	hostProxySocket := fmt.Sprintf("/lxg/run/user/%s/lxg.bus", ctx.UID)
	_, err := os.Stat(hostProxySocket)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("start dbus socket error: %w", err)
	} else if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}

	// Path to LXG D-Bus router service
	routerSocket := fmt.Sprintf("/run/user/%s/lxg.router", ctx.UID)

	// Check if router is already active and responding
	conn, err := net.DialTimeout("unix", routerSocket, 200*time.Millisecond)
	if err == nil {
		conn.Close()
		return nil
	}

	// Router is not responding, remove any stale socket
	err = os.Remove(routerSocket)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove dbus socket error: %w", err)
	}

	// Find lxg binary
	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = "/usr/local/bin/lxg"
	}

	// Spawn router as a detached background daemon
	cmd := exec.Command(binaryPath, "container", "router")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start dbus router error: %w", err)
	}

	// Release process resources so it runs independently
	go func() {
		_ = cmd.Wait()
	}()

	// Wait up to 2 seconds for router socket to become ready
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		conn, err := net.DialTimeout("unix", routerSocket, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
	}

	return nil
}

// StartRouter starts the D-Bus router in foreground
func StartRouter(ctx *command.Context) error {
	signalCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	router := NewRouter(ctx)
	return router.Start(signalCtx)
}
