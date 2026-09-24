package setup

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mateussouzaweb/lxg/command"
)

//go:embed scripts/*.sh
var scriptsFS embed.FS

// InstallInitScript on container to perform LXG initialization
func InstallInitScript(ctx *command.Context) error {

	destination := "/etc/profile.d/99-lxg.sh"
	content, err := scriptsFS.ReadFile("scripts/init.sh")
	if err != nil {
		return fmt.Errorf("install script read error: %w", err)
	}

	err = os.MkdirAll(filepath.Dir(destination), 0755)
	if err != nil {
		return fmt.Errorf("install script mkdir error: %w", err)
	}

	err = os.WriteFile(destination, content, 0755)
	if err != nil {
		return fmt.Errorf("install script write error: %w", err)
	}

	return nil
}
