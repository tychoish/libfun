package libfun

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
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
	send := pipe.Handler()

	return pipe.Generator().
		PreHook(fun.Worker(
			func(ctx context.Context) error {
				return filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
					if err != nil {
						return fs.SkipAll
					}

					out, err := fn(p, d)
					if err != nil {
						ec.When(!ers.Is(err, fs.SkipDir, fs.SkipAll), err)
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
		).Stream().WithHook(func(st *fun.Stream[T]) { st.AddError(ec.Resolve()) })
}
