// Package itertools is an implementation of Python's itertools
// using the proposed Go range-over-func: https://go.dev/issue/61405
//
// This is for experimentation, not real use.
package itertools

import (
	"testing"

	"github.com/neild/itertools/internal/coro"
)

func init() {
	if !testing.Testing() {
		panic("please don't try to actually use this")
	}
}

type Iter[T any] func(func(T) bool) bool

type Iter2[T1, T2 any] func(func(T1, T2) bool) bool

type Integer interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~int
}

type Number interface {
	Integer | ~float32 | ~float64
}

// Range yields every value in [start, end).
func Range[T Integer](start, end T) Iter[T] {
	return func(yield func(T) bool) bool {
		for n := start; n < end; n++ {
			if !yield(n) {
				return false
			}
		}
		return true
	}
}

// Enumerate yields (0, p0), (1, p1), (2, p2), ...
func Enumerate[T any](it Iter[T]) Iter2[int, T] {
	return func(yield func(int, T) bool) bool {
		i := 0
		for v := range it {
			if !yield(i, v) {
				return false
			}
			i++
		}
		return true
	}
}

// FromSlice yields every element in s.
func FromSlice[T any](s []T) Iter[T] {
	return func(yield func(T) bool) bool {
		for _, v := range s {
			if !yield(v) {
				return false
			}
		}
		return true
	}
}

// Count yields start, start+1, start+2, ...
func Count[T Number](start T) Iter[T] {
	return CountBy[T](start, 1)
}

// CountBy yields start, start+step, start+2*step, ...
func CountBy[T Number](start, step T) Iter[T] {
	return func(yield func(T) bool) bool {
		n := start
		for n >= start {
			if !yield(n) {
				return false
			}
			n += step
		}
		return true
	}
}

// Repeat yields elem, elem, elem, ... endlessly.
func Repeat[T any](elem T) Iter[T] {
	return func(yield func(T) bool) bool {
		for yield(elem) {
		}
		return false
	}
}

// Repeat yields elem, elem, elem, ... up to n times.
func RepeatN[T any](elem T, n int) Iter[T] {
	return func(yield func(T) bool) bool {
		for n > 0 {
			if !yield(elem) {
				return false
			}
			n--
		}
		return true
	}
}

// Accumulate yields p0, f(p0, p1), f(f(p0, p1), p2), ...
func Accumulate[T any](it Iter[T], f func(a, b T) T) Iter[T] {
	return func(yield func(T) bool) bool {
		var acc T
		for i, v := range Enumerate(it) {
			if i == 0 {
				acc = v
			} else {
				acc = f(acc, v)
			}
			if !yield(acc) {
				return false
			}
		}
		return true
	}
}

// Chain yields every element in iters[0], followed by every element in iters[1], etc.
func Chain[T any](iters ...Iter[T]) Iter[T] {
	return ChainFromIter[T](FromSlice(iters))
}

// ChainFromIter yields every element produced by each input iterator.
func ChainFromIter[T any](iters Iter[Iter[T]]) Iter[T] {
	return func(yield func(T) bool) bool {
		for it := range iters {
			for v := range it {
				if !yield(v) {
					return false
				}
			}
		}
		return true
	}
}

// Pull converts a push iterator to a pull iterator.
// Taken directly from https://research.swtch.com/coro.
func Pull[V any](push Iter[V]) (pull func() (V, bool), stop func()) {
	copush := func(more bool, yield func(V) bool) V {
		if more {
			push(yield)
		}
		var zero V
		return zero
	}
	resume, _ := coro.New(copush)
	pull = func() (V, bool) {
		return resume(true)
	}
	stop = func() {
		resume(false)
	}
	return pull, stop
}

// Compress yields (d[0] if s[0]), (d[1] if s[1]), ...
func Compress[T any](data Iter[T], selectors Iter[bool]) Iter[T] {
	return func(yield func(T) bool) bool {
		pulldata, stopdata := Pull[T](data)
		defer stopdata()
		pullsel, stopsel := Pull[bool](selectors)
		defer stopsel()
		for {
			d, ok := pulldata()
			if !ok {
				return true
			}
			s, ok := pullsel()
			if !ok {
				return true
			}
			if s && !yield(d) {
				return false
			}
		}
		return true
	}
}

// DropWhile yields seq[n], seq[n+1], ..., starting when pred returns false.
func DropWhile[T any](seq Iter[T], pred func(T) bool) Iter[T] {
	return func(yield func(T) bool) bool {
		ok := false
		for v := range seq {
			if !ok {
				if pred(v) {
					continue
				}
				ok = true
			}
			if !yield(v) {
				return false
			}
		}
		return true
	}
}

// FilterFalse yields elements of seq where pred(elem) is false.
func FilterFalse[T any](seq Iter[T], pred func(T) bool) Iter[T] {
	return func(yield func(T) bool) bool {
		for v := range seq {
			if pred(v) {
				continue
			}
			if !yield(v) {
				return false
			}
		}
		return true
	}
}

// GroupBy returns an iteration over consecutive keys and groups from seq.
//
// The keyFunc computes a key for each element.
// GroupBy yields (key K, groupIter Iter[E]) for each contiguous
// group of elements in seq with the same key.
//
// The groupIter iterator is only valid until the GroupBy iterator produces another value.
func GroupBy[K comparable, E any](seq Iter[E], keyFunc func(E) K) Iter2[K, Iter[E]] {
	return func(yield func(K, Iter[E]) bool) bool {
		pull, stop := Pull[E](seq)
		defer stop()
		val, ok := pull()
		if !ok {
			return true
		}
		key := keyFunc(val)
		for ok {
			advance := true
			groupIter := func(yield func(E) bool) bool {
				for {
					if !yield(val) {
						return false
					}
					val, ok = pull()
					if !ok {
						return true
					}
					if k := keyFunc(val); k != key {
						advance = false
						key = k
						return true
					}
				}
			}
			if !yield(key, groupIter) {
				return false
			}
			if advance {
				// Advance to the next group.
				for range groupIter {
				}
			}
		}
		return true
	}
}

// Slice yields elements from [start,end).
func Slice[T any](seq Iter[T], start, end int) Iter[T] {
	return func(yield func(T) bool) bool {
		for i, v := range Enumerate(seq) {
			if i < start {
				continue
			}
			if i >= end {
				break
			}
			if !yield(v) {
				return false
			}
		}
		return true
	}
}

// Pairwise yields (p0, p1), (p1, p2), (p2, p3), ...
func Pairwise[T any](seq Iter[T]) Iter2[T, T] {
	return func(yield func(T, T) bool) bool {
		first := true
		var last T
		for v := range seq {
			if !first && !yield(last, v) {
				return false
			}
			first = false
			last = v
		}
		return true
	}
}

// itertools.starmap doesn't translate well to Go.

// TakeWhile yields seq[0], seq[1], ..., until pred returns false.
func TakeWhile[T any](seq Iter[T], pred func(T) bool) Iter[T] {
	return func(yield func(T) bool) bool {
		for v := range seq {
			if !pred(v) {
				break
			}
			if !yield(v) {
				return false
			}
		}
		return true
	}
}

// Tee splits one iterator into n.
//
// The original iterator must not be used after the Tee call.
//
// It is not safe to access the resulting iterators from
// multiple goroutines at the same time.
func Tee[T any](seq Iter[T], n int) []Iter[T] {
	next, stop := Pull(seq)
	type chunk struct {
		size int
		done bool
		vals [64]T
		next *chunk
	}
	head := &chunk{}

	nextFromChunk := func(c *chunk, i int) (T, int, *chunk, bool) {
		var zero T
		if i >= c.size && c.size >= len(c.vals) && !c.done {
			if c.next == nil {
				c.next = &chunk{}
			}
			c = c.next
			i = 0
		}
		if i >= c.size {
			if c.done {
				return zero, 0, nil, false
			}
			var ok bool
			c.vals[c.size], ok = next()
			if !ok {
				c.done = true
				return zero, 0, nil, false
			}
			c.size++
		}
		return c.vals[i], i + 1, c, true
	}

	iters := make([]Iter[T], n)
	running := n
	for nn := range n {
		c := head
		iters[nn] = func(yield func(T) bool) bool {
			defer func() {
				running--
				if running == 0 {
					stop()
				}
			}()
			i := 0
			for {
				var v T
				var ok bool
				v, i, c, ok = nextFromChunk(c, i)
				if !ok {
					return true
				}
				if !yield(v) {
					return false
				}
			}
		}
	}
	return iters
}
