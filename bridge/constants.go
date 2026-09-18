package bridge

// Max request size
const maxRequestSize = 64 * 1024

// Path to LXG socket
const socketPath = "/run/user/1000/lxg.sock"

// User ID used on both host and container
const allowedUID = 1000

// Allowed commands to run on host
var allowedCommands = []string{
	"echo",
	"xdg-open",
	"gio",
}
