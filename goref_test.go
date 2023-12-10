package ref

import (
	"context"
	"testing"
)

func TestRef(t *testing.T) {
	t.Parallel()
	testCases := [...]struct {
		name        string
		refValue    any
		updateValue any
	}{
		{
			name:        "ValidRef",
			refValue:    0,
			updateValue: 10,
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			testRef := NewRef(tc.refValue)
			v := testRef.Value()
			if v != tc.refValue {
				t.Errorf("unexpected ref value, expected %d got %d", tc.refValue, v)
				return
			}
			testRef.SetValue(tc.updateValue)
			v = testRef.Value()
			if v != tc.updateValue {
				t.Errorf("unexpected ref value, expected %d got %d", tc.updateValue, v)
			}
		})
	}
}

func TestWatch(t *testing.T) {
	t.Parallel()
	const maxExecutions = 100
	countExecutionsWatch := 0
	testRef := NewRef(struct{}{})
	readyCh := make(chan struct{})

	Watch(func(actualValues, prevValues []struct{}, ctx context.Context) {
		countExecutionsWatch++
		readyCh <- struct{}{}
	}, testRef)

	for i := 0; i < maxExecutions; i++ {
		<-readyCh
		testRef.SetValue(struct{}{})
	}
	<-readyCh

	// plus-one because the Watch function executes f before returning
	if countExecutionsWatch != maxExecutions+1 {
		t.Errorf("unexpected count value, expected %d got %d", maxExecutions+1, countExecutionsWatch)
		return
	}

	t.Run("CancelFunc", func(t *testing.T) {

		const setUntil = 50
		countExecutionsWatch = 0

		testRef := NewRef(struct{}{})

		readyCh := make(chan struct{})
		var chAlreadyClosed bool

		cancel := Watch(func(actualValues, prevValues []struct{}, ctx context.Context) {
			countExecutionsWatch++
			if !chAlreadyClosed {
				readyCh <- struct{}{}
			}
		}, testRef)
		<-readyCh

		for i := 0; i < maxExecutions; i++ {
			if i >= setUntil {
				cancel()
				if !chAlreadyClosed {
					close(readyCh)
					chAlreadyClosed = true
				}
			}
			testRef.SetValue(struct{}{})
			<-readyCh
		}

		if countExecutionsWatch != setUntil+1 {
			t.Errorf("unexpected count value, expected %d got %d", setUntil+1, countExecutionsWatch)
			return
		}
	})
}
