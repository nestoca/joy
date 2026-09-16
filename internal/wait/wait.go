package wait

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/davidmdm/x/xerr"
)

type eventuallyOptions struct {
	interval  time.Duration
	timeout   time.Duration
	tickLabel string
}

type EventuallyOption func(*eventuallyOptions)

func WithInterval(interval time.Duration) EventuallyOption {
	return func(opts *eventuallyOptions) { opts.interval = interval }
}

func WithTimeout(timeout time.Duration) EventuallyOption {
	return func(opts *eventuallyOptions) { opts.timeout = timeout }
}

func WithTicker(label string) EventuallyOption {
	return func(opts *eventuallyOptions) { opts.tickLabel = label }
}

func Eventually(ctx context.Context, fn func(ctx context.Context) error, opts ...EventuallyOption) (err error) {
	options := eventuallyOptions{
		interval: time.Second,
		timeout:  10 * time.Second,
	}

	for _, apply := range opts {
		apply(&options)
	}

	if label := options.tickLabel; label != "" {
		defer Tick(ctx, label)()
	}

	ctx, cancel := context.WithTimeout(ctx, options.timeout)
	defer cancel()

	var (
		timer = time.NewTimer(0)
		i     = 0
	)
	for {
		select {
		case <-ctx.Done():
			return xerr.Join(err, context.Cause(ctx))
		case <-timer.C:
			i++
			err = fn(ctx)
			if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			timer.Reset(options.interval)
		}
	}
}

func Tick(ctx context.Context, label string) func() {
	fmt.Fprint(os.Stderr, label+" ")
	ticker := time.NewTicker(time.Second)

	ctx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})

	stop := func() {
		cancel()
		<-done
	}

	go func() {
		defer close(done)
		defer fmt.Fprint(os.Stderr, "\n")
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Fprint(os.Stderr, ".")
			}
		}
	}()

	return stop
}

func TickFunc(ctx context.Context, label string, fn func(ctx context.Context) error) error {
	defer Tick(ctx, label)()
	return fn(ctx)
}
