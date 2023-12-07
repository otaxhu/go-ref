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

	_, ready := Watch(func(prevValues []struct{}, ctx context.Context) {
		countExecutionsWatch++
	}, testRef)
	<-ready

	for i := 0; i < maxExecutions; i++ {

		testRef.SetValue(struct{}{})
		<-ready
	}

	// plus-one because the Watch function executes f before returning
	if countExecutionsWatch != maxExecutions+1 {
		t.Errorf("unexpected count value, expected %d got %d", maxExecutions+1, countExecutionsWatch)
		return
	}

	t.Run("CancelFunc", func(t *testing.T) {

		const setUntil = 50
		countExecutionsWatch = 0

		testRef := NewRef(struct{}{})

		cancel, ready := Watch(func(prevValues []struct{}, ctx context.Context) {
			countExecutionsWatch++
		}, testRef)
		<-ready

		for i := 0; i < maxExecutions; i++ {
			if i >= setUntil {
				cancel()
			}
			testRef.SetValue(struct{}{})
			<-ready
		}

		if countExecutionsWatch != setUntil+1 {
			t.Errorf("unexpected count value, expected %d got %d", setUntil+1, countExecutionsWatch)
			return
		}
	})
}
