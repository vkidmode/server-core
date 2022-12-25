package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type WriteLog interface {
	Log(ctx context.Context, message string)
}

type Core interface {
	Launch(ctx context.Context) error
	AddRunner(in Runner)
}

type core struct {
	runnableStack            chan recoverWrapper // stack with jobs need to run
	errorStack               chan error          // stack with errors
	logger                   WriteLog            // you can use this logger for custom logging
	cancel                   context.CancelFunc  // context.Cancel func
	timeout                  time.Duration       // time when forced termination will happen after crushing
	workersCount             uint8               // count of currently running workers
	workerGracefulStopSignal chan interface{}    // the channel with all launch signals
}

// NewCore logger is optional field, can be nil
func NewCore(logger WriteLog, timeout time.Duration, runnersCount uint8) Core {
	return &core{
		runnableStack:            make(chan recoverWrapper, runnersCount),
		errorStack:               make(chan error, runnersCount),
		workerGracefulStopSignal: make(chan interface{}, runnersCount),
		logger:                   logger,
		timeout:                  timeout,
	}
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

	defer c.wait()

	for {
		select {
		case err := <-c.errorStack:
			if err != nil {
				c.stop()
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *core) wait() {
	timeout, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	for {
		select {
		case <-timeout.Done():
			return
		default:
			if c.workersCount == 0 {
				return
			}
			c.readStopSignal()
		}
	}
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
	c.incWorkersCount()

	err := wrapper.run(ctx)
	if err == nil || !strings.Contains(err.Error(), panicError.Error()) {
		c.genStopSignal()
		return err
	}

	c.logger.Log(ctx, fmt.Sprintf("panic happened: %s", err.Error()))

	c.runnableStack <- wrapper
	return nil
}

func (c *core) readStopSignal() {
	<-c.workerGracefulStopSignal
	c.workersCount--
}

func (c *core) incWorkersCount() {
	c.workersCount++
}

func (c *core) genStopSignal() {
	c.workerGracefulStopSignal <- nil
}
