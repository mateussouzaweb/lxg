package container

import "os"

// IsContainer checks if terminal context is from container or not
func IsContainer() (bool, error) {

	// Container has the LXG_CONTAINER environment variable
	if os.Getenv("LXG_CONTAINER") == "1" {
		return true, nil
	}

	// Container mounts host /run folder on /lxg/run
	_, err := os.Stat("/lxg/run")
	if err != nil && !os.IsNotExist(err) {
		return false, err
	} else if err != nil && os.IsNotExist(err) {
		return false, nil
	} else if err == nil {
		return true, nil
	}

	return false, nil
}
