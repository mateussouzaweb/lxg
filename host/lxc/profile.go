package lxc

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mateussouzaweb/lxg/command"
)

//go:embed profiles/*.yml
var profileSpecFS embed.FS

// SetupProfile on LXC installation
func SetupProfile(ctx *command.Context) (bool, error) {

	// Check if profile exists
	// Exit code 1 means profile not exists
	args := []string{"profile", "show", ctx.Profile}
	cmd := exec.Command("lxc", args...)
	cmd.Stdin = os.Stdin

	result, err := command.Execute(cmd)
	if err != nil {
		return false, fmt.Errorf("profile show error: %w", err)
	} else if result.ExitCode > 1 {
		return false, fmt.Errorf("profile show error: %s", result.Error)
	} else if result.ExitCode == 0 {
		return false, nil
	}

	// Read profile specs
	profilePath := fmt.Sprintf("profiles/%s.yml", ctx.Profile)
	profileSpec, err := profileSpecFS.ReadFile(profilePath)
	if err != nil {
		return false, fmt.Errorf("profile read error: %w", err)
	}

	// Define UID on profile
	profileSpec = bytes.ReplaceAll(
		profileSpec,
		[]byte(":UID"),
		[]byte(ctx.UID),
	)

	// Define user on profile
	profileSpec = bytes.ReplaceAll(
		profileSpec,
		[]byte(":USER"),
		[]byte(ctx.User),
	)

	// Create profile from specs
	args = []string{"profile", "create", ctx.Profile}
	stdin := bytes.NewBuffer(profileSpec)
	cmd = exec.Command("lxc", args...)
	cmd.Stdin = stdin

	result, err = command.Execute(cmd)
	if err != nil {
		return false, fmt.Errorf("profile create error: %w", err)
	} else if result.ExitCode >= 1 {
		return false, fmt.Errorf("profile create error: %s", result.Error)
	}

	return true, nil
}

// AttachProfile to LXC container
func AttachProfile(ctx *command.Context) (bool, error) {

	// Check if profile already is attached to instance
	args := []string{"profile", "show", ctx.Profile}
	cmd := exec.Command("lxc", args...)
	cmd.Stdin = os.Stdin

	result, err := command.Execute(cmd)
	if err != nil {
		return false, fmt.Errorf("profile show error: %w", err)
	} else if result.ExitCode >= 1 {
		return false, fmt.Errorf("profile show error: %s", result.Error)
	} else if strings.Contains(result.Output, ctx.Container) {
		return false, nil
	}

	// Attach profile when not detected
	args = []string{"profile", "add", ctx.Container, ctx.Profile}
	cmd = exec.Command("lxc", args...)
	cmd.Stdin = os.Stdin

	result, err = command.Execute(cmd)
	if err != nil {
		return false, fmt.Errorf("profile add error: %w", err)
	} else if result.ExitCode >= 1 {
		return false, fmt.Errorf("profile add error: %s", result.Error)
	}

	return true, nil
}
