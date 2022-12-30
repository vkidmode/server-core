package core

import (
	"os"
	"time"
)

type Option func(o *options)

type options struct {
	gracefulSignals      []os.Signal
	gracefulDelay        *time.Duration
	enabledRerunWhenErrs bool
}

func WithEnabledRerunWhenErrs() Option {
	return func(o *options) {
		o.enabledRerunWhenErrs = true
	}
}

func WithGraceful(delay time.Duration, signals ...os.Signal) Option {
	return func(o *options) {
		o.gracefulDelay = &delay
		for _, signal := range signals {
			o.gracefulSignals = append(o.gracefulSignals, signal)
		}
	}
}
