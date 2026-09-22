package env

import "os"

// IsContainer checks if terminal context is from container or not
func IsContainer() bool {
	return os.Getenv("LXG_CONTAINER") == "1"
}

// IsIsolated checks if the container is running in isolated mode
func IsIsolated() bool {
	return os.Getenv("LXG_ISOLATED") == "1"
}

// IsIntegrated checks if the container is running in integrated mode
func IsIntegrated() bool {
	return os.Getenv("LXG_INTEGRATED") == "1"
}

// IsPrivileged checks if the container is running in privileged mode
func IsPrivileged() bool {
	return os.Getenv("LXG_PRIVILEGED") == "1"
}
