package ref

import "github.com/elliotchance/orderedmap/v2"

type Ref[T any] struct {
	value T

	// The key is a pointer to a function so it cannnot subscribe to the
	// same effect more than once.
	//
	// The value is empty struct so the value don't occupy memory
	effectsSubscribedTo *orderedmap.OrderedMap[*func(), struct{}]
}

func NewRef[T any](initialValue T) Ref[T] {
	return Ref[T]{
		value:               initialValue,
		effectsSubscribedTo: orderedmap.NewOrderedMap[*func(), struct{}](),
	}
}

func (r *Ref[T]) Value() T {
	return r.value
}

func (r *Ref[T]) SetValue(value T) {

	r.value = value

	var f func()
	for el := r.effectsSubscribedTo.Front(); el != nil; el = el.Next() {
		f = *el.Key
		f()
	}
}

// Watch executes the function f every time its dependencies are updated,
// the f function always executes at least once.
//
// Returns a cancel func to stop tracking the Reactive deps. If a call
// to cancel was already made, next calls will be no-op
func Watch[T any](f func(), deps ...Ref[T]) func() {
	if f == nil {
		return func() {}
	}
	for _, d := range deps {
		d.effectsSubscribedTo.Set(&f, struct{}{})
	}
	f()
	alreadyCanceled := false
	return func() {
		if alreadyCanceled {
			return
		}
		alreadyCanceled = true
		for _, d := range deps {
			d.effectsSubscribedTo.Delete(&f)
		}
	}
}
