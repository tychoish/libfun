package libfun

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/erc"
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

func RunCommand(ctx context.Context, cmd string) *fun.Stream[string] {
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer

	err := jasper.Context(ctx).
		CreateCommand(ctx).
		Append(cmd).
		SetOutputWriter(util.NewLocalBuffer(&stdoutBuf)).
		SetErrorWriter(util.NewLocalBuffer(&stderrBuf)).
		Run(ctx)

	if err != nil {
		return fun.MakeGenerator(func() (string, error) {
			return "", erc.Join(err, &ErrOutput{Cmd: cmd, Err: stderrBuf.String(),
				Out: stdoutBuf.String()})
		}).Stream()
	}

	return fun.MAKE.Lines(&stdoutBuf)
}

func RunCommandWithInput(ctx context.Context, cmd string, stdin io.Reader) *fun.Stream[string] {
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
		return fun.MakeGenerator(func() (string, error) {
			return "", erc.Join(err, &ErrOutput{Cmd: cmd, Err: stderrBuf.String(),
				Out: stdoutBuf.String()})
		}).Stream()
	}

	return fun.MAKE.Lines(&stdoutBuf)
}
