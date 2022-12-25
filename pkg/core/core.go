package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type WriteLog interface {
	Log(ctx context.Context, message string)
}

type Core interface {
	Launch(ctx context.Context) error
	AddRunner(in Runner)
}

// NewCore logger is optional field, can be nil
func NewCore(logger WriteLog) Core {
	return &core{
		runnableStack: make(chan recoverWrapper, 10),
		errorStack:    make(chan error, 10),
		logger:        logger,
	}
}

type core struct {
	runnableStack chan recoverWrapper
	errorStack    chan error
	logger        WriteLog
	cancel        context.CancelFunc
}

func (c *core) AddRunner(in Runner) {
	c.runnableStack <- newRecoverWrapper(in)
}

func (c *core) Launch(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go c.waitForInterruption()

	go func(*core) {
		for stackItem := range c.runnableStack {
			item := stackItem
			go func(recoverWrapper) {
				c.errorStack <- c.rerunIfPanic(ctx, item)
			}(item)
		}
	}(c)

	select {
	case err := <-c.errorStack:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return nil
	}
	return nil
}

func (c *core) waitForInterruption() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	c.stop()
}

func (c *core) stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *core) rerunIfPanic(ctx context.Context, wrapper recoverWrapper) error {
	err := wrapper.run(ctx)
	if err == nil || !strings.Contains(err.Error(), panicError.Error()) {
		return err
	}

	c.logger.Log(ctx, fmt.Sprintf("panic happened: %s", err.Error()))

	c.runnableStack <- wrapper
	return nil
}
