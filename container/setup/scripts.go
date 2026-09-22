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

// InstallScripts on container
func InstallScripts(ctx *command.Context) error {

	list := map[string]string{
		"/usr/local/bin/gio":       "scripts/gio.sh",
		"/usr/local/bin/xdg-open":  "scripts/xdg-open.sh",
		"/etc/profile.d/99-lxg.sh": "scripts/init.sh",
	}

	for destination, source := range list {

		content, err := scriptsFS.ReadFile(source)
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

	}

	return nil
}
