package core

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

type WriteLog interface {
	Log(ctx context.Context, message string)
}

type Core interface {
	Launch(ctx context.Context) error
	AddRunner(in Runner, runnerTypeWorker bool)
}

type core struct {
	runnableStack            chan recoverWrapper // stack with jobs need to run
	errorStack               chan error          // stack with errors
	logger                   WriteLog            // you can use this logger for custom logging
	cancel                   context.CancelFunc  // context.Cancel func
	timeout                  time.Duration       // time when forced termination will happen after crushing
	workersCount             int32               // count of currently running workers
	workerGracefulStopSignal chan interface{}    // the channel with all launch signals
	runnerTypeWorker         int32
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

func (c *core) AddRunner(in Runner, runnerTypeWorker bool) {
	c.runnableStack <- newRecoverWrapper(in, runnerTypeWorker)
}

func (c *core) Launch(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go c.waitForInterruption()

	go func(*core) {
		for stackItem := range c.runnableStack {
			item := stackItem
			go func(recoverWrapper) {
				if item.runnerTypeWorker {
					atomic.AddInt32(&c.runnerTypeWorker, 1)
				}

				c.errorStack <- c.rerunIfPanic(ctx, item)

				if item.runnerTypeWorker {
					atomic.AddInt32(&c.runnerTypeWorker, -1)
				}
				if c.runnerTypeWorker == 0 {
					cancel()
				}
			}(item)
		}
	}(c)

	defer c.waitGraceful()

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

func (c *core) waitGraceful() {
	timeout, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	for {
		if c.workersCount == 0 {
			return
		}
		c.readStopSignal(timeout)
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

	if c.logger != nil {
		c.logger.Log(ctx, fmt.Sprintf("panic happened: %s", err.Error()))
	}

	time.Sleep(100 * time.Millisecond)
	c.runnableStack <- wrapper
	time.Sleep(100 * time.Millisecond)
	c.decWorkersCount()
	return nil
}

func (c *core) readStopSignal(timeoutCtx context.Context) {
	select {
	case <-timeoutCtx.Done():
		atomic.StoreInt32(&c.workersCount, 0)
	case <-c.workerGracefulStopSignal:
		c.decWorkersCount()
	}
}

func (c *core) incWorkersCount() {
	atomic.AddInt32(&c.workersCount, 1)
}

func (c *core) decWorkersCount() {
	atomic.AddInt32(&c.workersCount, -1)
}

func (c *core) genStopSignal() {
	c.workerGracefulStopSignal <- nil
}
