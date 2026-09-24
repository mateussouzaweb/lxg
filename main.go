package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/mateussouzaweb/lxg/command"
	"github.com/mateussouzaweb/lxg/container"
	"github.com/mateussouzaweb/lxg/container/dbus"
	"github.com/mateussouzaweb/lxg/help"
	"github.com/mateussouzaweb/lxg/host"
	"github.com/mateussouzaweb/lxg/host/lxc"
)

// Extract value from args
func extract(args []string, value string) []string {
	index := slices.Index(args, value)
	return append(args[:index], args[index+1:]...)
}

// Handle command based on args
func handle(args []string) error {

	// Extract command
	name := args[0]
	args = extract(args, name)

	// Check if command was called from host or container
	// Containers has the LXG_CONTAINER environment variable
	// On container setup, special --from-host flag is passed
	containerEnv := os.Getenv("LXG_CONTAINER") == "1"
	fromHostFlag := slices.Contains(args, "--from-host")
	insideContainer := containerEnv || fromHostFlag

	// Block execution based on context
	hostOnly := []string{"listen", "setup", "start", "stop", "restart", "run"}
	containerOnly := []string{"container"}

	if insideContainer && slices.Contains(hostOnly, name) {
		return fmt.Errorf("method can be run only on host")
	} else if !insideContainer && slices.Contains(containerOnly, name) {
		return fmt.Errorf("method can be run only on container")
	}

	// Handle commands
	switch name {
	case "help", "--help", "-h":
		ctx := command.NewContext(args)
		return help.Print(ctx)

	case "listen":
		ctx := command.NewContext(args)
		return host.Init(ctx)

	case "setup":
		ctx := command.NewContext(args)
		ctx = command.WithContainer(ctx)
		return host.Setup(ctx)

	case "start":
		ctx := command.NewContext(args)
		ctx = command.WithContainer(ctx)
		return lxc.StartContainer(ctx)

	case "stop":
		ctx := command.NewContext(args)
		ctx = command.WithContainer(ctx)
		return lxc.StopContainer(ctx)

	case "restart":
		ctx := command.NewContext(args)
		ctx = command.WithContainer(ctx)
		return lxc.RestartContainer(ctx)

	case "run":
		ctx := command.NewContext(args)
		ctx = command.WithContainer(ctx)
		return lxc.RunCommand(ctx)

	case "container":
		if len(args) == 0 {
			return fmt.Errorf("missing command for container")
		}

		subcommand := args[0]
		args = extract(args, subcommand)

		switch subcommand {
		case "setup":
			ctx := command.NewContext(args)
			return container.Setup(ctx)
		case "init":
			ctx := command.NewContext(args)
			return container.Init(ctx)
		case "router":
			ctx := command.NewContext(args)
			return dbus.StartRouter(ctx)
		case "request":
			ctx := command.NewContext(args)
			return container.BridgeRequest(ctx)
		}

		return fmt.Errorf("unknown container command: %s", subcommand)
	}

	return fmt.Errorf("unknown command: %s", name)
}

func main() {

	// Ensure minimal arguments
	if len(os.Args) < 2 {
		help.Print(&command.Context{})
		os.Exit(2)
	}

	// Handle command
	err := handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err.Error())
		os.Exit(1)
	}

}
