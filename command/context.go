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
	User      string
	UID       string
	Args      []string
}

// NewContext creates a new context with given args
func NewContext(args []string) *Context {

	uid := fmt.Sprintf("%d", os.Getuid())
	user := os.Getenv("USER")

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
		Args:      args,
	}
}
