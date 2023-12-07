package ref

import (
	"context"
	"sync"

	"github.com/elliotchance/orderedmap/v2"
)

var keyCancelFunc = struct{}{}

type Ref[T any] struct {
	value T

	// The key is a pointer to a function so it cannnot subscribe to the
	// same effect more than once.
	watchersSuscribedTo *orderedmap.OrderedMap[
		*Watcher[T],
		[]*Ref[T],
	]

	// The context in which the Done channel will be closed (a call to a cancel function
	// provided by context.WithCancel) if a new value is set to Ref
	ctxSuscribedTo map[*Watcher[T]]context.Context

	// This channel sends a signal that indicates that the execution of f function (whether
	// called from Watch or SetValue) has returned and the program is synchronized with the return
	// of f
	readyChSuscribedTo chan struct{}

	mu sync.RWMutex
}

func NewRef[T any](initialValue T) *Ref[T] {
	return &Ref[T]{
		value: initialValue,

		watchersSuscribedTo: orderedmap.NewOrderedMap[
			*Watcher[T],
			[]*Ref[T],
		](),

		ctxSuscribedTo: map[*Watcher[T]]context.Context{},
	}
}

// Returns the value of Ref
func (r *Ref[T]) Value() T {
	defer r.mu.RUnlock()
	r.mu.RLock()
	return r.value
}

// Sets a new value to Ref, cancels all the contexts that was suscribed to, and
// executes all the watchers that was suscribed to (in that specific order).
//
// IMPORTANT NOTE:
//
// In order to achieve a reactive behaviour and to perform the cancelation of the
// contexts at the same time the watchers are running, the watchers are executed in a
// different goroutine, not synchronized with the return of SetValue.
//
// In order to synchronize the return of the watcher and the return of SetValue,
// the caller needs to await the ready channel (returned by Watch function) just right
// after a call to SetValue
//
// Example:
//
//	reactiveCount := ref.NewRef(0)
//	stop, ready := ref.Watch(func(prevValues []int, ctx context.Context) {
//	    fmt.Println("Previous values:", prevValues)
//	    fmt.Println("Actual values:", reactiveCount.Value())
//	}, reactiveCount)
//
//	<-ready // The actual goroutine will wait until the watcher is executed
//
//	reactiveCount.SetValue(reactiveCount.Value() + 1)
//
//	<-ready // Waiting on the same channel for the watcher to be executed again
func (r *Ref[T]) SetValue(value T) {
	r.mu.RLock()
	for _, v := range r.ctxSuscribedTo {
		callCancelFunc(v)
	}
	r.mu.RUnlock()

	defer r.mu.Unlock()
	r.mu.Lock()
	prev := r.value

	r.value = value

	for el := r.watchersSuscribedTo.Front(); el != nil; el = el.Next() {
		var prevValues []T

		for _, ref := range el.Value {
			if ref == r {
				prevValues = append(prevValues, prev)
				continue
			}
			prevValues = append(prevValues, ref.value)
		}
		ctx := ctxWithCancelFuncValue()

		r.ctxSuscribedTo[el.Key] = ctx
		go func(f func([]T, context.Context), ch chan struct{}) {
			f(prevValues, ctx)
			if ch != nil {
				ch <- struct{}{}
			}
		}(*el.Key, r.readyChSuscribedTo)
	}
}

var watchMutex sync.Mutex

type Watcher[T any] func(prevValues []T, ctx context.Context)

// Watch executes the watcher every time its dependencies are updated,
// the watcher always executes at least once. The first time is executed
// prevValues is nil, next calls prevValues will be a slice with lenght len(deps)
// preserving the order of the dependencies with the previous values of the
// dependecies (previous value of the dependency updated, the rest of them will be the actual values)
//
// Watch returns a "stop" function to stop tracking the Refs deps, and a channel that signals
// the caller that the watcher has already return, see Ref's SetValue docs for more info about
// this. If a call to stop was already made, next calls will be no-op
func Watch[T any](watcher Watcher[T], deps ...*Ref[T]) (stop func(), ready <-chan struct{}) {
	defer watchMutex.Unlock()
	watchMutex.Lock()

	readyCh := make(chan struct{})

	ctx := ctxWithCancelFuncValue()

	for _, d := range deps {
		d.watchersSuscribedTo.Set(&watcher, deps)
		d.ctxSuscribedTo[&watcher] = ctx
		d.readyChSuscribedTo = readyCh
	}
	go func() {
		watcher(nil, ctx)
		if readyCh != nil {
			readyCh <- struct{}{}
		}
	}()
	alreadyCanceled := false
	mu1 := sync.Mutex{}
	return func() {
		defer mu1.Unlock()
		mu1.Lock()
		if alreadyCanceled {
			return
		}
		alreadyCanceled = true
		for _, d := range deps {
			d.watchersSuscribedTo.Delete(&watcher)
			callCancelFunc(d.ctxSuscribedTo[&watcher])
		}
		if readyCh != nil {
			close(readyCh)
		}
	}, readyCh
}
