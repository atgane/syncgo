package ds

import (
	"sync/atomic"
)

type Closer struct {
	ch     chan struct{}
	closed atomic.Bool
}

func NewCloser() *Closer {
	c := &Closer{
		ch: make(chan struct{}),
	}
	return c
}

func (c *Closer) Wait() {
	<-c.ch
}

func (c *Closer) Close() {
	c.closed.CompareAndSwap(false, true)
	close(c.ch)
}

func (c *Closer) Done() <-chan struct{} {
	return c.ch
}

func (c *Closer) Closed() bool {
	return c.closed.Load()
}
