## Reactive primitives in Go
Library for reactive values in idiomatic Go

### Installation:
1. Execute this command on your terminal:
```bash
$ go get github.com/otaxhu/go-ref
```
2. Import this library with the name `ref`, given that `go-ref` is an invalid import name

### Quick Usage:
```go
package main

import (
    "context"

    "github.com/otaxhu/go-ref"
)

func main() {
    // optionally you can specify the generic type
    count := ref.NewRef[int](0)

    ref.Watch[int](func(actualValues, prevValues []int, ctx context.Context) {
        fmt.Println("the actual value of count is:", actualValues[0])
        fmt.Println("the previous value of count was:", prevValues[0])
    }, count)

    // Some event that triggers the change of count
    if userClicked {
        count.SetValue(count.Value() + 1)
    }
}
```
You can also use the `ctx` variable inside of the WatcherFunc to pass it to functions that performs cancelation logic, this cancel signal is send only if there is a new value setted to the Ref
