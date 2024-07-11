package main

import (
	"context"
	"fmt"
	"time"

	"github.com/vkidmode/server-core/pkg/core"
)

func main() {
	ctx := context.Background()

	app := core.NewCore(nil, 5*time.Second, 10)
	app.AddRunner(runner1, true)
	app.AddRunner(runner2, true)
	err := app.Launch(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func runner1(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("runner 1 graceful stop")
			return nil
		default:
			fmt.Println("panic happened")
			panic("some shit happened")
		}
	}
}

func runner2(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("runner 2 graceful stop")
			return nil
		default:
			time.Sleep(1 * time.Second)
			fmt.Println("runner 2 is working")
		}
	}
}
