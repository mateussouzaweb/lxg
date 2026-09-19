package bridge

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"slices"

	"github.com/mateussouzaweb/lxg/command"
)

// CreateRequest to perform command on host, via socket
func CreateRequest(ctx *command.Context) error {

	args := ctx.Args
	if len(args) == 0 {
		return fmt.Errorf("missing command to execute on host")
	}

	// Extract command
	command := args[0]
	index := slices.Index(args, command)
	args = append(args[:index], args[index+1:]...)

	// Extract --no-wait flag
	noWaitFlag := "--no-wait"
	waitCmd := !slices.Contains(args, noWaitFlag)
	if slices.Contains(args, noWaitFlag) {
		index := slices.Index(args, noWaitFlag)
		args = append(args[:index], args[index+1:]...)
	}

	// Send request on socket and wait for response
	request := NewRequest()
	request.Command = command
	request.Args = args
	request.Wait = waitCmd

	// Connect to LXG socket
	socketPath := fmt.Sprintf("/run/user/%s/lxg.sock", ctx.UID)
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("connect to host error: %w", err)
	}

	defer conn.Close()

	err = json.NewEncoder(conn).Encode(request)
	if err != nil {
		return fmt.Errorf("send request error: %w", err)
	}

	response := NewResponse()
	err = json.NewDecoder(conn).Decode(&response)
	if err != nil {
		return fmt.Errorf("receive response error: %w", err)
	}
	if !response.OK {
		return fmt.Errorf("response error: %s", response.Message)
	}

	os.Exit(response.ExitCode)
	return nil
}
