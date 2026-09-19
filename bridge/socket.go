package bridge

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/mateussouzaweb/lxg/command"
)

// Init socket listener to receive communication
func InitSocket(ctx *command.Context, cancel context.Context) error {

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

	// Close listener when context is cancelled
	go func() {
		<-cancel.Done()
		listener.Close()
	}()

	fmt.Printf("Bridge socket listening on %s\n", socketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			fmt.Printf("Bridge socket error: %s\n", err)
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
				fmt.Printf("Bridge socket error: %s\n", err)
			}
		}(conn)
	}

	fmt.Printf("Bridge socket closed.\n")
	return nil
}
