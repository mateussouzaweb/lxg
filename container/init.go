package container

import (
	"github.com/mateussouzaweb/lxg/context"
)

// Init LXG environment support
func Init(ctx *context.Context) error {

	err := InitEnvironment(ctx)
	if err != nil {
		return err
	}

	return nil
}
