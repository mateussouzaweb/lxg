package bridge

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// WriteJSON response on connection
func WriteJSON(conn net.Conn, response *Response) error {
	return json.NewEncoder(conn).Encode(response)
}

// WriteError response on connection
func WriteError(conn net.Conn, err error) error {
	response := NewResponse()
	response.OK = false
	response.PID = 0
	response.ExitCode = 1
	response.Message = err.Error()
	return WriteJSON(conn, response)
}

// HandleRequest on connection
func HandleRequest(conn net.Conn) error {

	dateTime := time.Now()
	fmt.Printf("Received new request: %s\n", dateTime.Format(time.RFC3339))

	// Check for authentication
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return errors.New("connection is not a unix socket")
	}

	rawConn, err := unixConn.SyscallConn()
	if err != nil {
		return fmt.Errorf("get socket error: %w", err)
	}

	var credentials *unix.Ucred
	var controlErr error
	err = rawConn.Control(func(fd uintptr) {
		credentials, controlErr = unix.GetsockoptUcred(
			int(fd),
			unix.SOL_SOCKET,
			unix.SO_PEERCRED,
		)
	})

	if err != nil {
		return fmt.Errorf("get peer credentials error: %w", err)
	}
	if controlErr != nil {
		return fmt.Errorf("read peer credentials error: %w", controlErr)
	}
	if credentials.Uid != allowedUID {
		return fmt.Errorf("authentication error: unauthorized uid: %d", credentials.Uid)
	}

	// Read request information
	deadline := time.Now().Add(10 * time.Second)
	err = conn.SetReadDeadline(deadline)
	if err != nil {
		return err
	}

	request := NewRequest()
	reader := io.LimitReader(conn, maxRequestSize)
	reader = bufio.NewReader(reader)
	err = json.NewDecoder(reader).Decode(&request)
	if err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}

	// Check if command can be executed
	command := strings.TrimSpace(request.Command)
	if command == "" {
		return errors.New("missing command")
	}
	if !slices.Contains(allowedCommands, command) {
		return fmt.Errorf("command not allowed: %s", command)
	}

	// The host process inherits the actual desktop environment
	cmd := exec.Command(command, request.Args...)
	cmd.Dir = os.Getenv("HOME")
	cmd.Env = os.Environ()
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("command error %s: %w", command, err)
	}

	// Run in background if don't need to wait
	// In this case, errors are not handled
	if !request.Wait {
		response := NewResponse()
		response.OK = true
		response.PID = cmd.Process.Pid
		response.ExitCode = 0
		response.Message = "async"

		go func() { cmd.Wait() }()
		return WriteJSON(conn, response)
	}

	// Wait for result and print it
	exitCode := 0
	err = cmd.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return err
		}
	}

	response := NewResponse()
	response.OK = exitCode == 0
	response.PID = cmd.Process.Pid
	response.ExitCode = exitCode
	response.Message = ""

	return WriteJSON(conn, response)
}
