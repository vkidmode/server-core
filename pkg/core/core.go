package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"time"
)

var errPanic = errors.New("panic")

type runner func(ctx context.Context) error

type Core struct {
	options       *options
	errorCallback func(error)
	panicCallback func(string)
}

func NewCore(opts ...Option) *Core {
	var options options
	for _, o := range opts {
		o(&options)
	}

	return &Core{
		options: &options,
	}
}

func (c *Core) SetErrorCallback(callback func(error)) {
	c.errorCallback = callback
}

func (c *Core) SetPanic(callback func(string)) {
	c.panicCallback = callback
}

func (c *Core) Launch(ctx context.Context, runners ...runner) {
	childCtx, childCancel := context.WithCancel(ctx)

	runnersChan := make(chan runner)
	go c.launch(childCtx, runnersChan)

	for _, r := range runners {
		runnersChan <- wrap(r)
	}

	wait := make(chan struct{})
	if len(c.options.gracefulSignals) > 0 {
		go func() {
			osSignal := make(chan os.Signal)
			signal.Notify(osSignal, c.options.gracefulSignals...)
			<-osSignal
			childCancel()
			if c.options.gracefulDelay != nil {
				time.Sleep(*c.options.gracefulDelay)
			}
			wait <- struct{}{}
		}()
	}

	go func() {
		<-ctx.Done()
		wait <- struct{}{}
	}()

	<-wait
	childCancel()
}

func (c *Core) launch(ctx context.Context, runnersChan chan runner) {
	for r := range runnersChan {
		go func(r runner) {
			if err := r(ctx); err != nil {
				if !errors.Is(err, errPanic) {
					if c.options.enabledRerunWhenErrs {
						runnersChan <- r
					}

					if c.errorCallback != nil {
						c.errorCallback(err)
					}
					return
				}

				if c.panicCallback != nil {
					c.panicCallback(errPanic.Error())
				}

				runnersChan <- r
			}
		}(r)
	}
}

func wrap(r runner) runner {
	return func(ctx context.Context) (err error) {
		defer func() {
			recoverErr := recover()
			if nil == recoverErr {
				return
			}

			var panicMsg string
			switch msg := recoverErr.(type) {
			case string:
				panicMsg = msg
			case []byte:
				panicMsg = string(msg)
			}

			err = fmt.Errorf("%w: %v %s", errPanic, fmt.Sprintf("%+v", panicMsg), string(debug.Stack()))
		}()

		return r(ctx)
	}
}
