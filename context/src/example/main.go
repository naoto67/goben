package main

import (
	"context"
	"fmt"
	"runtime"
)

func main() {
	fmt.Println(runtime.NumGoroutine())
	ctx1 := context.Background()
	ctx2, f := context.WithCancel(ctx1)
	ctx3, f := context.WithCancel(ctx2)
	fmt.Println(ctx2)
	fmt.Println(ctx3)

	// f()

	fmt.Println(ctx2)
	fmt.Println(ctx3)

	_, _ = context.WithCancel(context.Background())

	fmt.Println(runtime.NumGoroutine())

	f()

	fmt.Println(runtime.NumGoroutine())

}
