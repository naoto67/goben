package main

import (
	"context"
	"fmt"
	"runtime"
	"time"
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

	fmt.Println("GOROUTINE: ", runtime.NumGoroutine())

	_, _ = context.WithCancel(context.Background())
	ctx4, _ := context.WithTimeout(ctx3, time.Hour)
	fmt.Println(ctx4)

	fmt.Println("GOROUTINE: ", runtime.NumGoroutine())

	f()

	fmt.Println("GOROUTINE: ", runtime.NumGoroutine())

}
