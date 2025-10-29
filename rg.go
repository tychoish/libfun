package libfun

import (
	"bytes"
	"context"
	"path/filepath"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/ft"
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
//
// The iterator only provides access to the fully qualified filenames
// not the contents of the operation.
func Ripgrep(ctx context.Context, jpm jasper.Manager, args RipgrepArgs) *fun.Stream[string] {
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

	dt.NewSlice(args.Types).ReadAll(func(t string) { cmd.PushMany("--type", t) })
	dt.NewSlice(args.ExcludedTypes).ReadAll(func(t string) { cmd.PushMany("--type-not", t) })

	ft.ApplyWhen(args.Invert, cmd.Push, "--invert-match")
	ft.ApplyWhen(args.IgnoreFile != "", cmd.AppendSlice, []string{"--ignore-file", args.IgnoreFile})
	ft.ApplyWhen(args.Zip, cmd.Push, "--search-zip")
	ft.ApplyWhen(args.WordRegexp, cmd.Push, "--word-regexp")
	cmd.PushMany("--regexp", args.Regexp)

	err := jpm.CreateCommand(ctx).
		Directory(args.Path).
		Add(cmd).
		SetOutputSender(level.Info, sender).
		SetErrorSender(level.Error, grip.Sender()).
		Run(ctx)

	iter := fun.Convert(fnx.MakeConverter(func(in string) string {
		in = filepath.Join(args.Path, in)
		if args.Directories {
			return filepath.Dir(in)
		}
		return in
	})).Stream(fun.MAKE.Lines(&buf))

	iter.AddError(err)

	if args.Unique {
		return itertool.Uniq(iter)
	}

	return iter
}
