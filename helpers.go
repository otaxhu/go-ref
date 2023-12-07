package ref

import "context"

func callCancelFunc(ctx context.Context) {
	if c, ok := ctx.Value(keyCancelFunc).(context.CancelFunc); ok {
		c()
	} else {
		panic("the cancel function key was deleted from the context")
	}
}

func ctxWithCancelFuncValue() context.Context {
	ctx := context.Background()
	ctx, c := context.WithCancel(ctx)
	ctx = context.WithValue(ctx, keyCancelFunc, c)
	return ctx
}
