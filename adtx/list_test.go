package adt

import (
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tychoish/fun/assert"
	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/fun/stw"
)

func TestAtomicList(t *testing.T) {
	t.Run("BasicSmoke", func(t *testing.T) {
		t.Run("Back", func(t *testing.T) {
			l := new(List[int])
			check.Equal(t, l.Len(), 0)
			l.PushBack(stw.Ptr(1))
			check.Equal(t, l.Len(), 1)
			l.PushBack(stw.Ptr(2))
			check.Equal(t, l.Len(), 2)
			// recall
			check.Equal(t, stw.Deref(l.Tail().Value()), 2)
			check.Equal(t, l.Len(), 2)
			check.NotNil(t, l.Head())
			check.NotNil(t, l.Tail())
			check.Equal(t, stw.Deref(l.PopBack()), 2)
			check.Equal(t, l.Len(), 1)
			check.NotNil(t, l.PopBack())
			check.Equal(t, l.Len(), 0)
		})
		t.Run("", func(t *testing.T) {
			l := new(List[int])
			check.Equal(t, l.Len(), 0)
			l.PushFront(stw.Ptr(1))
			check.Equal(t, l.Len(), 1)
			l.PushFront(stw.Ptr(2))
			check.Equal(t, l.Len(), 2)
			// recall
			check.Equal(t, l.Head().Ref(), 2)
			check.Equal(t, l.Len(), 2)
			check.NotNil(t, l.Head())
			check.NotNil(t, l.Tail())
			check.Equal(t, stw.Deref(l.PopFront()), 2)
			check.Equal(t, l.Len(), 1)
			check.NotNil(t, l.PopFront())
			check.Equal(t, l.Len(), 0)
		})
	})
	t.Run("PopEmptyList", func(t *testing.T) {
		t.Run("Back", func(t *testing.T) {
			l := new(List[int])
			check.Equal(t, l.Len(), 0)
			check.NotPanic(t, func() { check.Nil(t, l.PopBack()) })
			check.Equal(t, l.Len(), 0)
			check.NotPanic(t, func() { check.Nil(t, l.PopBack()) })
		})
		t.Run("Front", func(t *testing.T) {
			l := new(List[int])
			check.Equal(t, l.Len(), 0)
			check.NotPanic(t, func() { check.Nil(t, l.PopFront()) })
			check.Equal(t, l.Len(), 0)
			check.NotPanic(t, func() { check.Nil(t, l.PopFront()) })
		})
		t.Run("ValueAccessorIsNilSafe", func(t *testing.T) {
			l := new(List[int])
			assert.True(t, !(l.head.Defined()))
			elem := l.Tail()
			assert.NotNil(t, elem)
			assert.Nil(t, elem.Value())
			assert.True(t, elem.root)
			assert.True(t, !(elem.Ok()))
		})
	})
	t.Run("Access", func(t *testing.T) {
		t.Run("Get", func(t *testing.T) {
			t.Run("NonNil", func(t *testing.T) {
				l := new(List[int])
				l.PushBack(stw.Ptr(42))
				val, ok := l.Head().Get()
				check.True(t, ok)
				check.Equal(t, 42, stw.Deref(val))
				vv, ok := l.Head().RefOk()
				check.Equal(t, vv, 42)
				check.True(t, ok)
			})
			t.Run("Nil", func(t *testing.T) {
				l := new(List[int])
				check.Equal(t, l.Len(), 0)
				elem := l.Head()
				check.NotNil(t, elem)
				check.Equal(t, l.Len(), 0)
				v, ok := elem.Get()
				check.Nil(t, v)
				check.True(t, !(ok))
				vv, ok := elem.RefOk()
				check.Zero(t, vv)
				check.True(t, !(ok))
			})
		})
		t.Run("Unset", func(t *testing.T) {
			l := new(List[int])
			l.PushBack(stw.Ptr(42))
			elem := l.Head()
			check.True(t, elem.Ok())
			v, ok := elem.Unset()
			check.True(t, ok)
			check.NotNil(t, v)
			check.Equal(t, stw.Deref(v), 42)
			v, ok = elem.Unset()
			check.True(t, !(ok))
			check.Nil(t, v)
		})
		t.Run("UnsetNil", func(t *testing.T) {
			l := new(List[int])
			check.Equal(t, l.Len(), 0)
			nada := l.Tail()
			check.NotNil(t, nada)
			check.True(t, nada.root)
			check.Nil(t, nada.Value())
			check.True(t, !(nada.Ok()))
			v, ok := nada.Unset()
			check.Nil(t, v)
			check.True(t, !(ok))
		})

		t.Run("Drop", func(t *testing.T) {
			l := new(List[int])
			l.PushBack(stw.Ptr(42))
			check.NotNil(t, l.Head())
			check.Equal(t, l.Len(), 1)
			check.NotNil(t, l.Head())
			check.Equal(t, l.Len(), 1)
			l.Head().Drop()
			check.Equal(t, l.Len(), 0)
			check.Nil(t, l.Head().Pop())
		})
	})
	t.Run("LockHandling", func(t *testing.T) {
		t.Run("Smoke", func(t *testing.T) {
			l := new(List[int])
			l.PushBack(stw.Ptr(4))
			l.PushBack(stw.Ptr(8))
			l.PushBack(stw.Ptr(16))
			l.PushBack(stw.Ptr(32))
			l.PushBack(stw.Ptr(64))
			mid := l.Head().Next().Next()
			check.Equal(t, stw.Deref(mid.Value()), 16)
			mid.mtx().Lock()
			sig := make(chan struct{})
			startSig := make(chan struct{})
			called := &atomic.Int64{}
			started := &atomic.Int64{}
			go func() {
				started.Add(1)
				close(startSig)
				defer close(sig)
				defer called.Add(1)
				l.Head().Next().append(l.makeElem(stw.Ptr(12)))
			}()
			runtime.Gosched()
			<-startSig
			time.Sleep(3 * time.Millisecond)
			check.Equal(t, started.Load(), 1)
			check.Equal(t, called.Load(), 0)
			check.Equal(t, l.Len(), 5)
			l.PushBack(stw.Ptr(128))
			check.Equal(t, l.Len(), 6)
			check.Equal(t, started.Load(), 1)
			check.Equal(t, called.Load(), 0)
			mid.mtx().Unlock()
			<-sig
			check.Equal(t, called.Load(), 1)
			check.Equal(t, l.Len(), 7)
		})
	})
	t.Run("AddingElements", func(t *testing.T) {
		t.Run("Parallel", func(t *testing.T) {
			t.Skip()
			l := new(List[int])
			wg := &sync.WaitGroup{}
			wg.Add(32)
			for range 32 {
				go func() {
					defer wg.Done()
					time.Sleep(time.Duration(50 + rand.Int64N(100*int64(time.Millisecond))))
					l.Append(stw.Ptr(rand.Int()))
				}()
			}
		})
		t.Run("Serial", func(t *testing.T) {
			t.Run("Smoke", func(t *testing.T) {
				l := new(List[int])
				for range 32 {
					l.Append(stw.Ptr(rand.Int()))
				}
			})
			t.Run("Correctness", func(t *testing.T) {
				l := new(List[int])
				for int := range 32 {
					l.Append(stw.Ptr(int))
				}
				for val := range l.Iterator() {
				}
			})
		})
	})
}
