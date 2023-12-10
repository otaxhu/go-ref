package ref

import (
	"context"
	"sync"
)

var keyCancelFunc = struct{}{}

type Ref[T any] struct {
	value T

	// The key is a pointer to a watcher so it cannnot subscribe to the
	// same watcher more than once.
	watchersSuscribedTo map[*WatcherFunc[T]][]*Ref[T]

	// The context in which the Done channel will be closed (a call to a cancel function
	// provided by context.WithCancel) if a new value is set to Ref
	ctxSuscribedTo map[*WatcherFunc[T]]context.Context

	mu sync.RWMutex
}

func NewRef[T any](initialValue T) *Ref[T] {
	return &Ref[T]{
		value: initialValue,

		watchersSuscribedTo: map[*WatcherFunc[T]][]*Ref[T]{},

		ctxSuscribedTo: map[*WatcherFunc[T]]context.Context{},
	}
}

// Returns the value of Ref
func (r *Ref[T]) Value() T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.value
}

// SetValue cancels all the contexts that was suscribed to, sets a new value to Ref, and
// executes in others goroutines all the watchers that was suscribed to (in that specific order).
//
// IMPORTANT:
//
// The watchers are executed in its own goroutine, this is important because you may not see
// see the execution of the watchers unless you block the caller goroutine
func (r *Ref[T]) SetValue(value T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, v := range r.ctxSuscribedTo {
		callCancelFunc(v)
	}
	prev := r.value

	r.value = value

	for watcher, refs := range r.watchersSuscribedTo {

		var (
			prevValues, actualValues []T
		)

		ctx := ctxWithCancelFuncValue()

		for _, ref := range refs {
			if ref == r {
				prevValues = append(prevValues, prev)
				actualValues = append(actualValues, ref.value)

				ref.ctxSuscribedTo[watcher] = ctx
				continue
			}
			ref.mu.Lock()
			defer ref.mu.Unlock()
			actualValues = append(actualValues, ref.value)
			prevValues = append(prevValues, ref.value)

			ref.ctxSuscribedTo[watcher] = ctx
		}
		go (*watcher)(actualValues, prevValues, ctx)
	}
}

type WatcherFunc[T any] func(actualValues, prevValues []T, ctx context.Context)

// Watch executes the watcher every time its dependencies are updated,
// the watcher always executes at least once and in its own goroutine. actualValues is a
// slice with lenght len(deps) and will always contain the actual values of the
// dependencies. The first time the watcher function is executed prevValues is nil
// and actualValues will contain the actual values, next calls prevValues will be a slice
// with lenght len(deps) preserving the order of the dependencies with the previous values
// of the dependecies (previous value of the dependency updated, the rest of them will be
// the actual values, same as actualValues values)
//
// Watch returns a stop function to stop tracking the Refs deps. If a call to stop was
// already made, next calls will be no-op.
//
// IMPORTANT:
//
// You should not call ref.Value() or ref.SetValue() methods on a ref dependency inside
// of the watcher, the reason is because it will result in a deadlock and in a recursive call to
// the watcher. You should use the actualValues and prevValues slices if you want to access
// the values. However, you can call those methods on a ref that is not a dependency of the
// watcher
func Watch[T any](watcher WatcherFunc[T], deps ...*Ref[T]) (stopFunc func()) {
	noDuplicates := map[*Ref[T]]struct{}{}
	for _, d := range deps {
		noDuplicates[d] = struct{}{}
	}
	if len(deps) != len(noDuplicates) {
		panic("go-ref: you may have passed the same Ref more than once, the watcher won't execute more than once for the same Ref")
	}

	ctx := ctxWithCancelFuncValue()

	var actualValues []T

	for _, d := range deps {
		d.mu.Lock()
		defer d.mu.Unlock()

		actualValues = append(actualValues, d.value)
		d.watchersSuscribedTo[&watcher] = deps
		d.ctxSuscribedTo[&watcher] = ctx
	}

	go watcher(actualValues, nil, ctx)

	return sync.OnceFunc(func() {
		for _, d := range deps {
			d.mu.Lock()
			defer d.mu.Unlock()

			delete(d.watchersSuscribedTo, &watcher)
			callCancelFunc(d.ctxSuscribedTo[&watcher])
			delete(d.ctxSuscribedTo, &watcher)
		}
	})
}
