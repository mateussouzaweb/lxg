package help

import (
	"embed"
	"fmt"

	"github.com/mateussouzaweb/lxg/context"
)

//go:embed help.man
var helpFS embed.FS

// Print help message
func Print(ctx *context.Context) error {

	content, err := helpFS.ReadFile("help.man")
	if err != nil {
		return err
	}

	fmt.Println(string(content))
	return nil
}
