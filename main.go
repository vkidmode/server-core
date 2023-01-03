package main

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/vkidmode/server-core/pkg/core"
)

func getRunnerWithID(id int, isPanic bool) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				fmt.Printf("runner %d ctx canceled\n", id)
				return nil
			default:
				time.Sleep(1 * time.Second)
				if isPanic {
					panic(fmt.Sprintf("punic runner %d", id))
				}

				fmt.Printf("runner %d is working\n", id)
			}
		}
	}
}

func runnerWithError(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("runner_with_error ctx canceled")
			return nil
		default:
			time.Sleep(2 * time.Second)
			return fmt.Errorf("some error from runner_with_error")
		}
	}
}

func main() {
	ctxTimeout, _ := context.WithTimeout(context.Background(), 10*time.Second)

	delay := 1 * time.Second
	c := core.NewCore(
		core.WithGracefulDelay(delay),
		core.WithGracefulOsSignals(syscall.SIGINT, syscall.SIGTERM),
	)

	c.SetPanic(func(s string) {
		fmt.Printf("callback - %s\n", s)
	})

	runner0 := getRunnerWithID(0, false)
	runner1 := getRunnerWithID(1, false)
	runner2 := getRunnerWithID(2, true)

	if err := c.Launch(ctxTimeout,
		runner0,
		runner1,
		runner2,
	); err != nil {
		fmt.Println(err)
	}

	fmt.Println("waiting 1 second...")
	time.Sleep(delay)

	ctxTimeout, _ = context.WithTimeout(context.Background(), 10*time.Second)
	if err := c.Launch(ctxTimeout,
		runner0,
		runner1,
		runnerWithError,
	); err != nil {
		fmt.Println("launch error", err)
	}

	fmt.Println("waiting 1 second...")
	time.Sleep(delay)
}
