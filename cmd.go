package libfun

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/send"
	"github.com/tychoish/jasper"
)

type ErrOutput struct {
	Cmd string
	Err string
	Out string
}

func (e *ErrOutput) Error() string {
	return fmt.Sprintf("cmd: %q; output: %q; error: %q", e.Cmd, e.Out, e.Err)
}

func RunCommand(ctx context.Context, cmd string) *fun.Iterator[string] {
	var stdoutBuf bytes.Buffer
	stdout := send.MakeBytesBuffer(&stdoutBuf)
	stdout.SetPriority(level.Info)
	stdout.SetName("stdout")

	var stderrBuf bytes.Buffer
	stderr := send.MakeBytesBuffer(&stderrBuf)
	stderr.SetPriority(level.Info)
	stderr.SetName("stderr")
	err := jasper.Context(ctx).
		CreateCommand(ctx).
		Append(cmd).
		SetOutputSender(level.Info, stdout).
		SetErrorSender(level.Info, stderr).
		Run(ctx)

	if err != nil {
		return fun.MakeProducer(func() (string, error) {
			return "", ers.Join(err, &ErrOutput{Cmd: cmd, Err: stderrBuf.String(),
				Out: stdoutBuf.String()})
		}).Iterator()
	}

	return fun.HF.Lines(&stdoutBuf)
}

func RunCommandWithInput(ctx context.Context, cmd string, stdin io.Reader) *fun.Iterator[string] {
	var stdoutBuf bytes.Buffer
	stdout := send.MakeBytesBuffer(&stdoutBuf)
	stdout.SetPriority(level.Info)
	stdout.SetName("stdout")

	var stderrBuf bytes.Buffer
	stderr := send.MakeBytesBuffer(&stderrBuf)
	stderr.SetPriority(level.Info)
	stderr.SetName("stderr")
	err := jasper.Context(ctx).
		CreateCommand(ctx).
		Append(cmd).
		SetOutputSender(level.Info, stdout).
		SetErrorSender(level.Info, stderr).
		SetInput(stdin).
		Run(ctx)

	if err != nil {
		return fun.MakeProducer(func() (string, error) {
			return "", ers.Join(err, &ErrOutput{Cmd: cmd, Err: stderrBuf.String(),
				Out: stdoutBuf.String()})
		}).Iterator()
	}

	return fun.HF.Lines(&stdoutBuf)
}
