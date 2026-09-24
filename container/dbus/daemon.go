package dbus

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mateussouzaweb/lxg/command"
)

func routerPaths(uid string) (socket string, pidFile string) {
	socket = fmt.Sprintf("/run/user/%s/lxg.router", uid)
	pidFile = socket + ".pid"
	return socket, pidFile
}

func pidAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil
}

func readPidFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid pid file")
	}
	return pid, nil
}

func routerRunning(socket string, pidFile string) bool {
	pid, err := readPidFile(pidFile)
	if err != nil || !pidAlive(pid) {
		return false
	}
	_, err = os.Stat(socket)
	return err == nil
}

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

	routerSocket, pidFile := routerPaths(ctx.UID)
	if routerRunning(routerSocket, pidFile) {
		return nil
	}

	err = os.Remove(routerSocket)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove dbus socket error: %w", err)
	}
	_ = os.Remove(pidFile)

	binaryPath, err := os.Executable()
	if err != nil {
		binaryPath = "/usr/local/bin/lxg"
	}

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

	waitErr := make(chan error, 1)
	go func() {
		waitErr <- cmd.Wait()
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-waitErr:
			if err != nil {
				return fmt.Errorf("dbus router exited: %w", err)
			}
			return fmt.Errorf("dbus router exited before becoming ready")
		default:
		}
		if routerRunning(routerSocket, pidFile) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	return fmt.Errorf("dbus router did not become ready")
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
