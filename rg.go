package libfun

import (
	"bytes"
	"context"
	"iter"
	"path/filepath"

	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/stw"
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
func Ripgrep(ctx context.Context, jpm jasper.Manager, args RipgrepArgs) (iter.Seq[string], error) {
	args.Path = util.TryExpandHomedir(args.Path)
	var buf bytes.Buffer
	sender := send.MakeBytesBuffer(&buf)
	sender.SetPriority(level.Info)
	sender.SetName("ripgrep")
	sender.SetErrorHandler(send.ErrorHandlerFromSender(grip.Sender()))

	cmd := stw.Slice[string]{
		"rg",
		"--files-with-matches",
		"--line-buffered",
		"--color=never",
		"--trim",
	}

	for ty := range irt.Slice(args.Types) {
		cmd.Extend(irt.Args("--type", ty))
	}
	for t := range irt.Slice(args.ExcludedTypes) {
		cmd.Extend(irt.Args("--type-not", t))
	}
	if args.Invert {
		cmd.Push("--invert-match")
	}
	if args.IgnoreFile != "" {
		cmd.Extend(irt.Args("--ignore-file", args.IgnoreFile))
	}
	if args.Zip {
		cmd.Push("--search-zip")
	}
	if args.WordRegexp {
		cmd.Push("--word-regexp")
	}
	cmd.Extend(irt.Args("--regexp", args.Regexp))

	err := jpm.CreateCommand(ctx).
		Directory(args.Path).
		Add(cmd).
		SetOutputSender(level.Info, sender).
		SetErrorSender(level.Error, grip.Sender()).
		Run(ctx)
	if err != nil {
		return nil, err
	}
	seq := irt.Convert(irt.ReadLines(&buf),
		func(in string) string {
			in = filepath.Join(args.Path, in)
			if args.Directories {
				return filepath.Dir(in)
			}
			return in
		},
	)

	if args.Unique {
		return irt.Unique(seq), nil
	}

	return seq, nil
}
