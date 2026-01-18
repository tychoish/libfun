package irtx

import (
	"context"
	"iter"
	"sync"
	"time"

	"github.com/tychoish/fun/irt"
)

func ShardByHash[T any](ctx context.Context, num int64, seq iter.Seq[T], getHash func(T) int64) iter.Seq[iter.Seq[T]] {
	shards := makeShards(getHash)
	shards.init(num)

	populate := once(func() {
		go func() {
			defer shards.closeAll()
			timer := time.NewTimer(0)
			defer timer.Stop()
		NEXT:
			for value := range seq {
			RETRY:
				for ch := range shards.getShardsFrom(getHash(value)) {
					timer.Reset(20 * time.Millisecond)
					select {
					case <-ctx.Done():
						return
					case ch <- value:
						continue NEXT
					case <-timer.C:
						continue RETRY
					}
				}
			}
		}()
	})

	return irt.Merge(irt.Index(shards.iterator()), func(shardID int, ch chan T) iter.Seq[T] {
		populate()
		return func(yield func(T) bool) {
			defer shards.closeIdx(shardID)

			for recvAndYield(ctx, ch, yield) {
				continue
			}
		}
	})
}

func isNilChan[T any](in chan T) bool { return in == nil }

type shardSet[T any] struct {
	mutex sync.RWMutex
	chans []chan T
	start int64
	count int64
	hashf func(T) int64
}

func makeShards[T any](hashf func(T) int64) *shardSet[T] { return &shardSet[T]{hashf: hashf} }
func (sh *shardSet[T]) lockRW() *sync.RWMutex            { return lockRW(sh.mtx()) }
func (sh *shardSet[T]) lockRR() *sync.RWMutex            { return lockRR(sh.mtx()) }
func (sh *shardSet[T]) mtx() *sync.RWMutex               { return &sh.mutex }
func (sh *shardSet[T]) getChans(size int) []chan T {
	return irt.Collect(irt.WhileOk(irt.Generate2(ntimes(size, sh.makeCh))))
}

func (sh *shardSet[T]) init(size int64) {
	sh.count, sh.start = size, size
	sh.chans = sh.getChans(int(size))
}

func (*shardSet[T]) makeCh() chan T         { return make(chan T) }
func (*shardSet[T]) closeCh(in chan T)      { close(in) }
func (sh *shardSet[T]) get(idx int) chan T  { return sh.chans[idx] }
func (sh *shardSet[T]) nonNil(idx int) bool { return sh.chans[idx] == nil }
func (sh *shardSet[T]) unset(idx int)       { sh.count--; sh.chans[idx] = nil }
func (sh *shardSet[T]) doClose(idx int)     { sh.closeCh(sh.get(idx)); sh.unset(idx) }

func (sh *shardSet[T]) closeIdx(idx int) {
	defer withRW(sh.lockRW())
	whencallwith(sh.nonNil(idx), sh.doClose, idx)
}

func (sh *shardSet[T]) withRLock(op func() bool) bool { defer withRR(sh.lockRR()); return op() }
func (sh *shardSet[T]) closeAll()                     { defer withRW(sh.lockRW()); irt.Apply(sh.iterator(), sh.closeCh) }
func (sh *shardSet[T]) iterator() iter.Seq[chan T]    { return irt.Remove(irt.Slice(sh.chans), isNilChan) }
func (sh *shardSet[T]) len() int64                    { return int64(len(sh.chans)) }
func (sh *shardSet[T]) atIndex(index int64) chan T    { return sh.chans[index] }

func (sh *shardSet[T]) getShardsFrom(index int64) iter.Seq[chan T] {
	return func(yield func(chan T) bool) {
		for idx, ct := index%sh.count, int64(0); ct < sh.count && idx < sh.len(); idx, ct = idx+1%sh.count, ct+1 {
			if sh.withRLock(func() bool {
				if ch := sh.atIndex(idx); ch == nil {
					return false
				} else {
					return !yield(ch)
				}
			}) {
				return
			}
		}
	}
}

func once(op func()) func()                  { return sync.OnceFunc(op) }
func withRW(mtx *sync.RWMutex)               { mtx.Unlock() }
func lockRW(mtx *sync.RWMutex) *sync.RWMutex { mtx.Lock(); return mtx }
func withRR(mtx *sync.RWMutex)               { mtx.RUnlock() }
func lockRR(mtx *sync.RWMutex) *sync.RWMutex { mtx.RLock(); return mtx }

func whencallwith[T any](cond bool, then func(T), arg T) {
	if cond {
		then(arg)
	}
}

func whencall(cond bool, then func()) {
	if cond {
		then()
	}
}

func ntimes[T any](times int, op func() T) func() (T, bool) {
	var count int
	return func() (zero T, _ bool) {
		if count >= times {
			return zero, false
		}
		count++
		return op(), true
	}
}

func recvAndYield[T any](ctx context.Context, ch <-chan T, yield func(T) bool) bool {
	select {
	case <-ctx.Done():
		return false
	case item, ok := <-ch:
		return ok && yield(item)
	}
}
