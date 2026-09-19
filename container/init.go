package container

import "github.com/mateussouzaweb/lxg/command"

// Init LXG environment support
func Init(ctx *command.Context) error {

	err := InitEnvironment(ctx)
	if err != nil {
		return err
	}

	return nil
}
