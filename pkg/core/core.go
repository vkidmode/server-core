package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"runtime/debug"
	"time"
)

var errPanic = errors.New("panic")

type runner func(ctx context.Context) error

type Core struct {
	options       *options
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

func (c *Core) SetPanic(callback func(string)) {
	c.panicCallback = callback
}

func (c *Core) Launch(ctx context.Context, runners ...runner) error {
	childCtx, childCancel := context.WithCancel(ctx)

	runnersChan := make(chan runner)
	runnerErrChan := c.launch(childCtx, runnersChan)

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
		childCancel()
		wait <- struct{}{}
	}()

	select {
	case <-wait:
	case err := <-runnerErrChan:
		childCancel()
		return err
	}

	return nil
}

func (c *Core) launch(ctx context.Context, runnersChan chan runner) chan error {
	runnerErrs := make(chan error)
	go func() {
		for r := range runnersChan {
			go func(r runner) {
				if err := r(ctx); err != nil {
					if !errors.Is(err, errPanic) {
						runnerErrs <- err
						return
					}

					if c.panicCallback != nil {
						c.panicCallback(err.Error())
					}

					runnersChan <- r
				}
			}(r)
		}
	}()

	return runnerErrs
}

func wrap(r runner) runner {
	return func(ctx context.Context) (err error) {
		defer func() {
			if recoverErr := recover(); recoverErr != nil {
				var panicMsg string
				switch msg := recoverErr.(type) {
				case string:
					panicMsg = msg
				case []byte:
					panicMsg = string(msg)
				}

				err = fmt.Errorf("%w: %v %s", errPanic, fmt.Sprintf("%+v", panicMsg), string(debug.Stack()))
				return
			}

			if err != nil {
				err = fmt.Errorf("%s: %w", runtime.FuncForPC(reflect.ValueOf(r).Pointer()).Name(), err)
			}
		}()

		return r(ctx)
	}
}
