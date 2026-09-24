package command

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

type Context struct {
	Container string
	Profile   string
	UID       string
	User      string
	Home      string
	Args      []string
}

// NewContext creates a new context with given args
func NewContext(args []string) *Context {

	uid := fmt.Sprintf("%d", os.Getuid())
	user := os.Getenv("USER")
	home := os.Getenv("HOME")

	profile := "integrated"
	profileFlags := []string{
		"--privileged",
		"--isolated",
		"--integrated",
	}

	for _, flag := range profileFlags {
		if slices.Contains(args, flag) {
			profile = strings.TrimPrefix(flag, "--")
			index := slices.Index(args, flag)
			args = append(args[:index], args[index+1:]...)
		}
	}

	return &Context{
		Container: "",
		Profile:   profile,
		UID:       uid,
		User:      user,
		Home:      home,
		Args:      args,
	}
}

// WithContainer extract and define the container target
// Defaults to ubuntu container name when not detected
func WithContainer(ctx *Context) *Context {

	container := "ubuntu"
	args := ctx.Args

	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			continue
		}

		container = arg
		index := slices.Index(args, container)
		args = append(args[:index], args[index+1:]...)
		break
	}

	ctx.Container = container
	ctx.Args = args
	return ctx
}
