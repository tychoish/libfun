package libfun

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/ft"
)

// FileExists provies a function that "reads" correctly with the
// usually required semantics for "does this file exist."
func FileExists(path string) bool { return ft.Not(os.IsNotExist(ft.IgnoreFirst(os.Stat(path)))) }

// WalkDirIterator provides an alternate fun.Iterator-based interface
// to filepath.WalkDir. The filepath.WalkDir runs in a go routnine,
// and calls a simpler walk function: where you can output an object,
// [in most cases a string of the path] but the function is generic.
//
// If the first value of the walk function is nil, then the item is
// skipped the walk will continue, otherwise--assuming that the error
// is non-nil, it is de-referenced and returned by the iterator.
func WalkDirIterator[T any](path string, fn func(p string, d fs.DirEntry) (*T, error)) *fun.Stream[T] {
	ec := &erc.Collector{}

	pipe := fun.Blocking(make(chan T))
	send := fnx.NewHandler(pipe.Send().Write)

	return fun.MakeStream(fnx.NewFuture(pipe.Receive().Read).
		PreHook(fnx.Worker(
			func(ctx context.Context) error {
				return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}

					out, err := fn(p, d)
					if err != nil {
						ec.If(!ers.Is(err, fs.SkipDir, fs.SkipAll), err)
						return err
					}
					if out == nil {
						return nil
					}
					return send(ctx, *out)
				})
			}).
			Operation(fun.MAKE.ErrorHandlerWithoutTerminating(ec.Push)).
			PostHook(pipe.Close).
			Go().Once(),
		)).WithHook(func(st *fun.Stream[T]) { st.AddError(ec.Resolve()) })
}

type FsWalkOptions struct {
	Path       string
	IgnoreMode *fs.FileMode
	OnlyMode   *fs.FileMode

	SkipPermissionErrors bool
	IgnorePrefix         string
	IncludePrefixes      []string
}

func hasAnyPrefix(str string, prefixes []string) bool {
	for prefix := range slices.Values(prefixes) {
		if strings.HasPrefix(str, prefix) {
			return true
		}
	}
	return false
}

func FsWalkStream[T any](opts FsWalkOptions, fn func(p string, d fs.DirEntry) (*T, error)) *fun.Stream[T] {
	ec := &erc.Collector{}

	pipe := fun.Blocking(make(chan T))
	send := pipe.Handler()

	if opts.IgnorePrefix != "" && strings.HasPrefix(opts.Path, opts.IgnorePrefix) && len(opts.Path) > 1 {
		opts.IgnorePrefix = opts.IgnorePrefix[len(opts.Path)-1:]
	}

	return pipe.Generator().
		PreHook(fun.Worker(
			func(ctx context.Context) error {
				return filepath.WalkDir(opts.Path, func(p string, d fs.DirEntry, err error) error {
					switch {
					case opts.IgnorePrefix != "" && strings.HasPrefix(p, opts.IgnorePrefix):
						return nil
					case opts.IgnoreMode != nil && ft.Ref(opts.IgnoreMode) == d.Type():
						return nil
					case err != nil && opts.SkipPermissionErrors && errors.Is(err, fs.ErrPermission):
						return nil
					case err != nil && ers.Is(fs.SkipAll, fs.SkipDir, ers.ErrCurrentOpAbort, ers.ErrCurrentOpSkip):
						return nil
					case err != nil:
						ec.Push(err)
						return err
					case opts.OnlyMode != nil && ft.Ref(opts.OnlyMode) == d.Type() && err == nil:
						fallthrough
					case len(opts.IncludePrefixes) > 0 && hasAnyPrefix(p, opts.IncludePrefixes):
						fallthrough
					default:
						out, err := fn(p, d)
						switch {
						case err == nil && out == nil:
							return nil
						case err == nil && out != nil:
							return send(ctx, *out)
						case err != nil && ers.Is(fs.SkipAll, fs.SkipDir, ers.ErrCurrentOpAbort, ers.ErrCurrentOpSkip):
							return nil
						default:
							ec.Push(err)
							return err
						}
					}
				})
			}).
			Operation(fun.MAKE.ErrorHandlerWithoutTerminating(ec.Push)).
			PostHook(pipe.Close).
			Go().Once(),
		).Stream().WithHook(func(st *fun.Stream[T]) { st.AddError(ec.Resolve()) })
}

type SymbolicLinks struct {
	Path      string
	Target    string
	Timestamp time.Time
}

func SymbolicLinkIterFunc(path string, entry fs.DirEntry) (*SymbolicLinks, error) {
	if entry.Type()&fs.ModeSymlink != 0 {
		return nil, nil
	}
	info, err := entry.Info()
	if err != nil {
		return nil, err
	}

	path = filepath.Join(path, entry.Name())

	one, err1 := os.Readlink(path)
	two, err2 := filepath.EvalSymlinks(path)
	if err := erc.Join(err1, err2); err != nil {
		return nil, err
	}

	if one != two {
		return nil, fmt.Errorf("symlink targets %q and %q do not match", one, two)
	}

	return &SymbolicLinks{
		Timestamp: info.ModTime(),
		Path:      path,
		Target:    one,
	}, nil
}
