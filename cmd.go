package libfun

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"

	"github.com/tychoish/fun/irt"
	"github.com/tychoish/jasper"
	"github.com/tychoish/jasper/util"
)

type ErrOutput struct {
	Cmd string
	Err string
	Out string
}

func (e *ErrOutput) Error() string {
	return fmt.Sprintf("cmd: %q; output: %q; error: %q", e.Cmd, e.Out, e.Err)
}

func RunCommand(ctx context.Context, cmd string) (iter.Seq[string], error) {
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	err := jasper.Context(ctx).
		CreateCommand(ctx).
		Append(cmd).
		SetOutputWriter(util.NewLocalBuffer(&stdoutBuf)).
		SetErrorWriter(util.NewLocalBuffer(&stderrBuf)).
		Run(ctx)
	if err != nil {
		return nil, err
	}

	return irt.ReadLines(&stdoutBuf), nil
}

func RunCommandWithInput(ctx context.Context, cmd string, stdin io.Reader) (iter.Seq[string], error) {
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	err := jasper.Context(ctx).
		CreateCommand(ctx).
		Append(cmd).
		SetOutputWriter(util.NewLocalBuffer(&stdoutBuf)).
		SetErrorWriter(util.NewLocalBuffer(&stderrBuf)).
		SetInput(stdin).
		Run(ctx)
	if err != nil {
		return nil, err
	}

	return irt.ReadLines(&stdoutBuf), nil
}
