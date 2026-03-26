package service

import (
	"context"
	"sync"
	"time"
)

func DelayedContext(ctx context.Context, delay time.Duration) (
	context.Context, context.CancelFunc,
) {
	delayed := &delayedContext{
		parent: ctx,
		delay:  delay,
		done:   make(chan struct{}),
	}
	canceled := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			select {
			case <-canceled:
				delayed.finish(context.Canceled)
			case <-time.After(delay):
				delayed.finish(ctx.Err())
			}
		case <-canceled:
			delayed.finish(context.Canceled)
		}
	}()
	return delayed, sync.OnceFunc(func() { close(canceled) })
}

type delayedContext struct {
	parent context.Context
	delay  time.Duration
	mu     sync.Mutex
	done   chan struct{}
	err    error
}

func (c *delayedContext) Deadline() (time.Time, bool) {
	deadline, hasDeadline := c.parent.Deadline()
	if hasDeadline {
		deadline = deadline.Add(c.delay)
	}
	return deadline, hasDeadline
}

func (c *delayedContext) Done() <-chan struct{} {
	return c.done
}

func (c *delayedContext) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *delayedContext) Value(key any) any {
	return c.parent.Value(key)
}

func (c *delayedContext) finish(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.err = err
	close(c.done)
}
