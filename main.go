package main

import (
	"context"
	"fmt"
	"time"

	"github.com/vkidmode/server-core/pkg/core"
)

func main() {
	ctx := context.Background()

	app := core.NewCore()
	app.AddRunner(runner1)
	app.AddRunner(runner2)
	app.AddRunner(runner2)
	app.AddRunner(runner2)
	app.AddRunner(runner2)
	err := app.Launch(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func runner1(ctx context.Context) error {
	for {
		fmt.Println("runner1 is running")
		time.Sleep(1 * time.Second)
	}
}

func runner2(ctx context.Context) error {
	for {
		fmt.Println("runner2 is running")
		time.Sleep(1 * time.Second)
	}
}
