package container

import (
	"fmt"
	"os"
	"os/user"

	"github.com/mateussouzaweb/lxg/context"
)

// EnsureUID for any user on container
func EnsureUID(ctx *context.Context) error {

	// User must be greater than 1000
	// Check for existing user
	currentUID := max(os.Getuid(), 1000)
	requiredUID := fmt.Sprintf("%v", currentUID)
	existing, err := user.LookupId(requiredUID)
	if err == nil {
		fmt.Printf("Found existing user for UID %s: %s\n", requiredUID, existing.Username)
		return nil
	}

	// Request username creation
	fmt.Printf("WARNING!\n")
	fmt.Printf("Container has no user with UID %s.\n", requiredUID)
	fmt.Printf("Make sure to create a new user with matching UID.\n")
	fmt.Printf("If applicable, add user to sudoers too!")

	return nil
}
