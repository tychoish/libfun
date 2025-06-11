package libfun

import (
	"context"
	"fmt"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/ft"
	"github.com/tychoish/godmenu"
)

type DMenuCommand struct {
	// Selections contains a list of options that are passed to
	// dmenu.
	Selections []string
	// NextHandle is takes the user's selection fro mthe list
	// above, and either produces an error, returns (nil, nil)
	// (thus breaking the operation,) or produces another
	// *DMenuCommand for the next "stage"
	NextHandle func(ctx context.Context, selection string) (*DMenuCommand, error)
	// Stage is a human readable name for this command.
	Stage string
	// Configuration is optional and defaults to the
	// godmenu.DMenuConfiguration default values *or* the value
	// provided by the previous level.
	Configuration *godmenu.DMenuConfiguration
}

const ErrUndefinedOperation ers.Error = "undefined operation"

// DMenu takes a *DMenuCommand operation and returns a Worker that
// when executed will call one or more DMenu selections and
// operations.
func DMenu(cmd *DMenuCommand) fun.Worker {
	cmd.Stage = ft.Default(cmd.Stage, "root")

	return func(ctx context.Context) (err error) {
		if cmd == nil {
			return ErrUndefinedOperation
		}

		for {
			if err != nil {
				return err
			}

			if cmd == nil && err == nil {
				return nil
			}

			cmd, err = cmd.wrap()(ctx)
		}
	}
}

func (cmd *DMenuCommand) nextID() string { return fmt.Sprint(cmd.Stage, ".next") }

func (cmd *DMenuCommand) wrap() fun.Generator[*DMenuCommand] {
	return func(ctx context.Context) (*DMenuCommand, error) {
		if len(cmd.Selections) == 0 {
			return nil, ers.Wrapf(ErrUndefinedOperation, "selections: %q", cmd.Stage)
		}
		if cmd.NextHandle == nil {
			return nil, ers.Wrapf(ErrUndefinedOperation, "handler: %q", cmd.Stage)
		}

		selection, err := godmenu.RunDMenu(ctx, godmenu.Options{
			Selections: cmd.Selections,
			DMenu:      cmd.Configuration,
		})
		if err != nil {
			return nil, ers.Wrap(err, cmd.Stage)
		}

		next, err := cmd.NextHandle(ctx, selection)
		if err != nil {
			return nil, ers.Wrap(err, cmd.Stage)
		}
		if next == nil {
			return nil, nil
		}

		next.Configuration = ft.Default(next.Configuration, cmd.Configuration)
		next.Stage = ft.DefaultFuture(next.Stage, next.nextID)

		return next, nil
	}
}

func MenuOperation(ctx context.Context, om map[string]fun.Worker, conf *godmenu.DMenuConfiguration) error {
	mp := dt.NewMap(om)
	keys, err := mp.Keys().Slice(ctx)
	if err != nil {
		return err
	}

	selection, err := godmenu.RunDMenu(ctx, godmenu.Options{Selections: keys, DMenu: conf})
	if err != nil {
		return err
	}

	op := mp.Get(selection)
	if op == nil {
		return ers.Wrap(ErrUndefinedOperation, selection)
	}

	return op.Run(ctx)
}
