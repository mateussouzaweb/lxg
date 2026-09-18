package container

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/mateussouzaweb/lxg/command"
	"github.com/mateussouzaweb/lxg/context"
)

// Install dependencies on container
func InstallDependencies(ctx *context.Context) error {

	// Packages mapping per distribution
	packages := map[string][]string{
		"apt": {
			"apt", "install", "-y",
			"mesa-utils", "wayland-utils", "dbus-x11", "x11-apps",
			"xdg-utils", "xdg-desktop-portal", "xdg-desktop-portal-gtk",
			"xdg-desktop-portal-kde", "kwayland-integration",
			"pipewire", "pipewire-pulse", "wireplumber", "pipewire-audio", "pipewire-bin",
			"gnome-keyring", "libsecret-1-0",
		},
		"dnf": {
			"dnf", "install", "-y",
			"glx-utils", "wayland-utils", "dbus-x11",
			"xdg-utils", "xdg-desktop-portal", "xdg-desktop-portal-gtk",
			"xdg-desktop-portal-kde", "kwayland-integration",
			"pipewire", "pipewire-pulseaudio", "wireplumber",
			"gnome-keyring", "libsecret",
		},
		"pacman": {
			"pacman", "-Syu", "--noconfirm",
			"mesa-utils", "wayland-utils", "dbus", "xorg-xclock",
			"xdg-utils", "xdg-desktop-portal", "xdg-desktop-portal-gtk",
			"xdg-desktop-portal-kde", "kwayland-integration",
			"pipewire", "pipewire-pulse", "wireplumber",
			"gnome-keyring", "libsecret",
		},
		"apk": {
			"apk", "add", "--no-cache",
			"mesa-utils", "wayland-utils", "dbus-x11",
			"xdg-utils", "xdg-desktop-portal", "xdg-desktop-portal-gtk",
			"xdg-desktop-portal-kde", "kwayland",
			"pipewire", "pipewire-pulse", "wireplumber",
			"gnome-keyring", "libsecret",
		},
	}

	for binary, args := range packages {

		// Check if command is available
		path, err := exec.LookPath(binary)
		if err == nil {
			return err
		} else if path == "" {
			continue
		}

		fmt.Printf("Installing packages via %s...\n", binary)

		// Update repositories if using APT
		if strings.HasSuffix(binary, "apt") {
			args := []string{"apt", "update"}
			cmd := exec.Command("sudo", args...)
			cmd.Stdin = os.Stdin

			result, err := command.Execute(cmd)
			if err != nil {
				return fmt.Errorf("dependency error: %w", err)
			} else if result.ExitCode >= 1 {
				return fmt.Errorf("dependency error: %s", result.Error)
			}
		}

		// Execute package installation via sudo
		cmd := exec.Command("sudo", args...)
		cmd.Stdin = os.Stdin

		result, err := command.Execute(cmd)
		if err != nil {
			return fmt.Errorf("dependency error: %w", err)
		} else if result.ExitCode >= 1 {
			return fmt.Errorf("dependency error: %s", result.Error)
		}

		fmt.Printf("All packages installed successfully!\n")
		return nil
	}

	return fmt.Errorf("no supported package manager found (apt, dnf, pacman, apk)")
}

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

	// Request username
	fmt.Printf("Container has no user with UID %s.\n", requiredUID)
	fmt.Printf("Enter a username to create: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("failed to read username from input")
	}

	username := strings.TrimSpace(scanner.Text())
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	// Create user on container
	cmd := exec.Command("sudo", "bash", "-c", fmt.Sprintf(`
		ID="%s"
		NAME="%s"

		if command -v useradd &>/dev/null; then
			sudo useradd -u "${ID}" -m -s /bin/bash "${NAME}"
			sudo usermod -aG wheel "${NAME}"
			sudo passwd "${NAME}"
		elif command -v adduser &>/dev/null; then
			sudo adduser -u "${ID}" -D -s /bin/bash "${NAME}"
			sudo adduser "${NAME}" wheel
			sudo passwd "${NAME}"
		else
			echo "Error: No suitable command found to create user."
			exit 1
		fi`, requiredUID, username),
	)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("Successfully created user.\n")
	return nil
}
