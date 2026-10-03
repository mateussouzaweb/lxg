package run

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// BindMount source file into target.
// Requires CAP_SYS_ADMIN privilege
func BindMount(source string, target string, cancel context.Context) error {

	// Check for source file presence
	_, err := os.Stat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Gracefully skip
		}
		return fmt.Errorf("check mount error: %w", err)
	}

	// Create base folder
	err = os.MkdirAll(filepath.Dir(target), 0700)
	if err != nil {
		return fmt.Errorf("mkdir mount error: %w", err)
	}

	// Best-effort unmount. We ignore the error because if it's not mounted,
	// the kernel returns syscall.EINVAL, which is completely fine here.
	_ = syscall.Unmount(target, 0)

	// Remove old target if exists
	err = os.Remove(target)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove mount error: %w", err)
	}

	// Create an empty file to act as the mount target
	err = os.WriteFile(target, []byte(""), 0600)
	if err != nil {
		return fmt.Errorf("write mount error: %w", err)
	}

	// Bind mount the file using the Linux syscall
	err = syscall.Mount(source, target, "none", syscall.MS_BIND, "")
	if err != nil {

		cleanupErr := os.Remove(target)
		if cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			return fmt.Errorf("bind mount failed (%w), and cleanup also failed: %v", err, cleanupErr)
		}

		// Return the original mount error
		return fmt.Errorf("bind mount error: %w", err)
	}

	fmt.Printf("Bind mount activated on %s\n", target)

	// Clean up when context is cancelled
	go func() {
		<-cancel.Done()
		syscall.Unmount(target, 0)
		os.Remove(target)
	}()

	return nil
}
