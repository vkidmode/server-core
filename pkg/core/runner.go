package core

import (
	"context"
	"fmt"
	"runtime/debug"
)

var (
	panicError = fmt.Errorf("panic happened")
)

type Runner func(ctx context.Context) error

func newRecoverWrapper(runnerItem Runner) recoverWrapper {
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
			err = fmt.Errorf("%w: %v\n%s", panicError, panicToString(recoverErr), string(debug.Stack()))
		}
		<-runnerCtx.Done()
	}()

	err = r.runner(runnerCtx)
	return
}

func panicToString(panicErr interface{}) string {
	if panicErr == nil {
		return ""
	}
	switch e := panicErr.(type) {
	case string:
		return e
	case []byte:
		return string(e)
	}
	return fmt.Sprintf("%+v", panicErr)
}
