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
	runners       chan runner
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
		runners: make(chan runner),
		options: &options,
	}
}

func (c *Core) Add(runners ...runner) {
	for _, r := range runners {
		c.runners <- wrap(r)
	}
}

func (c *Core) Launch(ctx context.Context) {
	childCtx, childCancel := context.WithCancel(ctx)
	runnersErrs := make(chan error)
	go func() {
		for r := range c.runners {
			go func(r runner) {
				if err := r(childCtx); err != nil {
					if !errors.Is(err, errPanic) {
						runnersErrs <- err
						return
					}

					if c.panicCallback != nil {
						c.panicCallback(errPanic.Error())
					}

					c.runners <- r
				}
			}(r)
		}
	}()

	osSignal := make(chan os.Signal)
	go func() {
		signal.Notify(osSignal, c.options.gracefulSignals...)
	}()

	for {
		select {
		case <-osSignal:
			childCancel()
			if c.options.gracefulDelay != nil {
				time.Sleep(*c.options.gracefulDelay)
			}
			return
		case <-ctx.Done():
			childCancel()
			return
		case err := <-runnersErrs:
			if c.errorCallback != nil {
				c.errorCallback(err)
			}
		}
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
