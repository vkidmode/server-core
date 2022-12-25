package core

import (
	"context"
	"fmt"
	"strings"
)

type Core interface {
	Launch(ctx context.Context) error
	AddRunner(in Runner)
}

func NewCore() Core {
	return &core{
		runnableStack: make(chan recoverWrapper, 10),
		errorStack:    make(chan error, 10),
	}
}

type core struct {
	runnableStack chan recoverWrapper
	errorStack    chan error
}

func (c *core) AddRunner(in Runner) {
	c.runnableStack <- newRecoverWrapper(in)
}

func (c *core) Launch(ctx context.Context) error {
	go func(*core) {
		for stackItem := range c.runnableStack {
			item := stackItem
			go func(recoverWrapper) {
				c.errorStack <- c.rerunIfPanic(ctx, item)
			}(item)
		}
	}(c)

	for err := range c.errorStack {
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *core) rerunIfPanic(ctx context.Context, wrapper recoverWrapper) error {
	err := wrapper.run(ctx)
	if err == nil || !strings.Contains(err.Error(), panicError.Error()) {
		if err != nil {
			fmt.Println(err)
		}
		return err
	}

	fmt.Println("panic happened")

	c.runnableStack <- wrapper
	return nil
}
