package help

import (
	"embed"
	"fmt"

	"github.com/mateussouzaweb/lxg/command"
)

//go:embed help.man
var helpFS embed.FS

// Print help message
func Print(ctx *command.Context) error {

	content, err := helpFS.ReadFile("help.man")
	if err != nil {
		return err
	}

	fmt.Printf("%s", string(content))
	return nil
}
