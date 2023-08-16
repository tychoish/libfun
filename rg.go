package libfun

import (
	"bytes"
	"context"
	"path/filepath"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/itertool"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/send"
	"github.com/tychoish/jasper"
	"github.com/tychoish/jasper/util"
)

type RipgrepArgs struct {
	Types         []string
	ExcludedTypes []string
	Regexp        string
	Path          string
	IgnoreFile    string
	Directories   bool
	Unique        bool
	Invert        bool
	Zip           bool
	WordRegexp    bool
}

// Ripgrep runs a ripgrep operation using the provided jasper
// returning an iterator of the full path names of any file that
// ripgrep finds that matches regexp provided.
func Ripgrep(ctx context.Context, jpm jasper.Manager, args RipgrepArgs) *fun.Iterator[string] {
	args.Path = util.TryExpandHomedir(args.Path)
	var buf bytes.Buffer
	sender := send.MakeBytesBuffer(&buf)
	sender.SetPriority(level.Info)
	sender.SetName("ripgrep")
	sender.SetErrorHandler(send.ErrorHandlerFromSender(grip.Sender()))

	cmd := dt.Slice[string]{
		"rg",
		"--files-with-matches",
		"--line-buffered",
		"--color=never",
		"--trim",
	}

	dt.Sliceify(args.Types).Observe(func(t string) { cmd.Append("--type", t) })
	dt.Sliceify(args.ExcludedTypes).Observe(func(t string) { cmd.Append("--type-not", t) })

	cmd.AppendWhen(args.Invert, "--invert-match")
	cmd.AppendWhen(args.IgnoreFile != "", "--ignore-file", args.IgnoreFile)
	cmd.AppendWhen(args.Zip, "--search-zip")
	cmd.AppendWhen(args.WordRegexp, "--word-regexp")

	cmd.Append("--regexp", args.Regexp)

	err := jpm.CreateCommand(ctx).
		Directory(args.Path).
		Add(cmd).
		SetOutputSender(level.Info, sender).
		SetErrorSender(level.Error, grip.Sender()).
		Run(ctx)

	iter := fun.Converter(func(in string) string {
		in = filepath.Join(args.Path, in)
		if args.Directories {
			return filepath.Dir(in)
		}
		return in
	}).Convert(fun.HF.Lines(&buf))
	iter.AddError(err)

	if args.Unique {
		return itertool.Uniq(iter)
	}

	return iter
}
