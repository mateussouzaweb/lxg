package context

type Context struct {
	Container string
	Username  string
	Args      []string
}

// NewContext creates a new context with given args
func NewContext(args []string) *Context {
	return &Context{
		Container: "",
		Username:  "",
		Args:      args,
	}
}
