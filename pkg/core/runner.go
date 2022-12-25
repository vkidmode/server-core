package core

import (
	"context"
	"fmt"
)

var (
	panicError = fmt.Errorf("panic happened")
)

type runner func(ctx context.Context) error

func newRecoverWrapper(runnerItem runner) recoverWrapper {
	return recoverWrapper{
		runner: runnerItem,
	}
}

type recoverWrapper struct {
	runner func(ctx context.Context) error
}

func (r *recoverWrapper) run(ctx context.Context) (err error) {
	runnerCtx, cancel := context.WithCancel(ctx)

	defer func() {
		cancel()
		if recoverErr := recover(); recoverErr != nil {
			err = fmt.Errorf("%w: %v", panicError, recoverErr.(string))
		}
		<-runnerCtx.Done()
	}()

	err = r.runner(runnerCtx)
	return
}
