package context

import (
	"fmt"
	"os"
)

type Context struct {
	Container string
	UID       string
	Args      []string
}

// NewContext creates a new context with given args
func NewContext(args []string) *Context {

	uid := fmt.Sprintf("%d", os.Getuid())

	return &Context{
		Container: "",
		UID:       uid,
		Args:      args,
	}
}
