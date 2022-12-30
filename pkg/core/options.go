package core

import (
	"os"
	"time"
)

type Option func(o *options)

type options struct {
	gracefulSignals []os.Signal
	gracefulDelay   *time.Duration
}

func WithGracefulDelay(delay time.Duration) Option {
	return func(o *options) {
		o.gracefulDelay = &delay
	}
}

func WithGracefulOsSignals(signals ...os.Signal) Option {
	return func(o *options) {
		o.gracefulSignals = make([]os.Signal, 0, len(signals))
		for _, signal := range signals {
			o.gracefulSignals = append(o.gracefulSignals, signal)
		}
	}
}
