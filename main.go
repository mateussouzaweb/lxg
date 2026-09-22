package main

import (
	"fmt"
	"os"
	"slices"

	"github.com/mateussouzaweb/lxg/bridge"
	"github.com/mateussouzaweb/lxg/command"
	"github.com/mateussouzaweb/lxg/container"
	"github.com/mateussouzaweb/lxg/help"
	"github.com/mateussouzaweb/lxg/lxc"
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
	// Need to block execution based on context
	isContainer, err := container.IsContainer()
	if err != nil {
		return err
	}

	hostOnly := []string{"listen", "setup", "start", "stop", "run"}
	containerOnly := []string{"container"}

	if isContainer && slices.Contains(hostOnly, name) {
		return fmt.Errorf("method can be run only on host")
	} else if !isContainer && slices.Contains(containerOnly, name) {
		return fmt.Errorf("method can be run only on container")
	}

	// Handle commands
	switch name {
	case "help", "--help", "-h":
		ctx := command.NewContext(args)
		return help.Print(ctx)
	case "listen":
		ctx := command.NewContext(args)
		return bridge.Init(ctx)
	case "setup", "start", "stop", "run":

		// Extract container name
		container := "ubuntu"
		if len(args) > 0 {
			container = args[0]
			args = extract(args, container)
		}

		// Create context
		ctx := command.NewContext(args)
		ctx.Container = container

		switch name {
		case "setup":
			return lxc.Setup(ctx)
		case "start":
			return lxc.StartContainer(ctx)
		case "stop":
			return lxc.StopContainer(ctx)
		case "run":
			return lxc.RunCommand(ctx)
		}

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
			return container.StartRouter(ctx)
		case "request":
			ctx := command.NewContext(args)
			return bridge.CreateRequest(ctx)
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
