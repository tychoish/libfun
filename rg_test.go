package libfun

import (
	"context"
	"testing"
	"time"

	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/jasper"
)

func TestRipgrep(t *testing.T) {
	start := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jpm := jasper.NewManager(jasper.ManagerOptionSet(
		jasper.ManagerOptions{
			ID:           t.Name(),
			Synchronized: true,
			MaxProcs:     64,
			// Tracker:      ft.Must(track.New(t.Name())),
		}))
	args := RipgrepArgs{
		Types:       []string{"go"},
		Regexp:      "go:generate",
		Path:        "/home/tychoish/neon/cloud",
		Directories: true,
		Unique:      true,
	}
	iter, err := Ripgrep(ctx, jpm, args)
	if err != nil {
		t.Error(err)
	}
	count := 0

	for val := range iter {
		count++
		grip.Info(val)
	}
	grip.Info(message.Fields{
		"path":  args.Path,
		"count": count,
		"dur":   time.Since(start),
	})
}
