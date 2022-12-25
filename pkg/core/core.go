package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

type Core interface {
	Launch(ctx context.Context) error
	AddRunner(in runner)
}

func NewCore() Core {
	return &core{
		runnableStack: make(chan recoverWrapper, 10),
	}
}

type core struct {
	runnableStack chan recoverWrapper
}

func (c *core) AddRunner(in runner) {
	c.runnableStack <- newRecoverWrapper(in)
}

func (c *core) Launch(ctx context.Context) error {
	eg, ctx := errgroup.WithContext(ctx)

	go func(*core) {
		for stackItem := range c.runnableStack {
			fmt.Println("here")
			eg.Go(c.rerunIfPanic(ctx, stackItem))
		}
	}(c)

	time.Sleep(1 * time.Second)

	return eg.Wait()
}

func (c *core) rerunIfPanic(ctx context.Context, wrapper recoverWrapper) func() error {
	return func() error {
		err := wrapper.run(ctx)
		if err == nil || !strings.Contains(err.Error(), panicError.Error()) {
			return err
		}

		// logger should be here

		c.runnableStack <- wrapper
		return nil
	}
}
