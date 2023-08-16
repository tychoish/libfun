package libfun

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/erc"
)

// WalkDirIterator provides an alternate fun.Iterator-based interface
// to filepath.WalkDir. The filepath.WalkDir runs in a go routnine,
// and calls a simpler walk function: where you can output an object,
// [in most cases a string of the path] but the function is generic.
//
// If the first value of the walk function is nil, then the item is
// skipped the walk will continue, otherwise--assuming that the error
// is non-nil, it is de-referenced and returned by the iterator.
func WalkDirIterator[T any](ctx context.Context, path string, fn func(p string, d fs.DirEntry) (*T, error)) *fun.Iterator[T] {
	ec := &erc.Collector{}

	pipe := fun.Blocking(make(chan T))

	init := fun.Operation(func(ctx context.Context) {
		send := pipe.Processor()

		_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return fs.SkipAll
			}

			out, err := fn(p, d)
			if err != nil {
				if !errors.Is(err, fs.SkipDir) && !errors.Is(err, fs.SkipAll) {
					ec.Add(err)
				}
				return err
			}
			if out == nil {
				return nil
			}

			return send(ctx, *out)
		})
	}).PostHook(pipe.Close).Launch().Once()

	return pipe.Producer().PreHook(init).IteratorWithHook(erc.IteratorHook[T](ec))
}
