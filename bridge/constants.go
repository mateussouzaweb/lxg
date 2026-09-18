package bridge

// Max request size
const maxRequestSize = 64 * 1024

// Allowed commands to run on host
var allowedCommands = []string{
	"echo",
	"xdg-open",
	"gio",
}
