package main

import (
	"context"
	"fmt"
	"time"

	"github.com/vkidmode/server-core/pkg/core"
)

func main() {
	ctx := context.Background()

	app := core.NewCore(nil, 50*time.Second, 10)
	app.AddRunner(runner1, true)  // this is finishing
	app.AddRunner(runner3, true)  // this is finishing
	app.AddRunner(runner2, false) // this is waiting
	err := app.Launch(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func runner1(ctx context.Context) error {
	fmt.Println("runner1: you should wait me")
	time.Sleep(5 * time.Second)
	fmt.Println("runner1: good job")
	return nil
}

func runner3(ctx context.Context) error {
	fmt.Println("runner3: you should wait me")
	time.Sleep(10 * time.Second)
	fmt.Println("runner3: good job")
	return nil
}

func runner2(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("runner 2 graceful stop")
			return nil
		}
	}
}
