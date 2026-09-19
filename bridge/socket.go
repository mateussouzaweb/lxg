package bridge

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mateussouzaweb/lxg/command"
)

// Init socket listener to receive communication
func InitSocket(ctx *command.Context) error {

	// Path to LXG socket
	socketPath := fmt.Sprintf("/run/user/%s/lxg.sock", ctx.UID)

	// Remove old socket if exists
	err := os.Remove(socketPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove old socket error: %w", err)
	}

	// Create socket listener
	err = os.MkdirAll(filepath.Dir(socketPath), 0700)
	if err != nil {
		return fmt.Errorf("create socket directory error: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("create socket listener error: %w", err)
	}

	defer listener.Close()
	defer os.Remove(socketPath)

	// Only user can manipulate socket
	err = os.Chmod(socketPath, 0600)
	if err != nil {
		return fmt.Errorf("chmod socket error: %w", err)
	}

	// Listen for termination signals to gracefully close the listener
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalChan)

	go func() {
		<-signalChan
		listener.Close()
	}()

	fmt.Printf("Bridge listening on %s\n", socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			fmt.Printf("Bridge error: %s\n", err)
			continue
		}

		// Handle connection on worker
		go func(conn net.Conn) {
			defer conn.Close()
			err := HandleRequest(ctx, conn)
			if err != nil {
				err = WriteError(conn, err)
			}
			if err != nil {
				fmt.Printf("Bridge error: %s\n", err)
			}
		}(conn)
	}

	fmt.Printf("Bridge closed\n")
	return nil
}
