package ds

import (
	"sync"
	"sync/atomic"
)

type Eventloop[T any] struct {
	queue         chan T
	handler       func(T)
	dispatchCount int
	closeCh       chan struct{}
	closed        atomic.Bool
	forceClosed   atomic.Bool
	state         atomic.Int32
}

func NewEventloop[T any](dispatchCount, queueSize int, handler func(T)) *Eventloop[T] {
	e := &Eventloop[T]{
		queue:         make(chan T, queueSize),
		handler:       handler,
		dispatchCount: dispatchCount,
		closeCh:       make(chan struct{}),
	}
	return e
}

func (e *Eventloop[T]) Run() {
	wg := sync.WaitGroup{}
	for range e.dispatchCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.dispatch()
		}()
	}
	wg.Wait()
}

func (e *Eventloop[T]) Send(event T) error {
	if e.closed.Load() {
		return ErrAlreadyClosedLoop
	}

	e.state.Add(1)
	defer e.state.Add(-1)

	select {
	case <-e.closeCh:
		return ErrAlreadyClosedLoop
	default:
	}

	select {
	case <-e.closeCh:
		return ErrAlreadyClosedLoop
	case e.queue <- event:
		return nil
	}
}

func (e *Eventloop[T]) Close() {
	if !e.closed.CompareAndSwap(false, true) {
		return
	}

	close(e.closeCh)
}

func (e *Eventloop[T]) ForceClose() {
	if !e.closed.CompareAndSwap(false, true) {
		return
	}

	close(e.closeCh)
	for e.state.Load() != 0 {
	}
	e.forceClosed.Store(true)
}

func (e *Eventloop[T]) dispatch() {
	for {
		select {
		case event := <-e.queue:
			if e.forceClosed.Load() {
				return
			}
			e.handler(event)
			continue
		default:
		}

		select {
		case event := <-e.queue:
			if e.forceClosed.Load() {
				return
			}
			e.handler(event)
		case <-e.closeCh:
			if e.state.Load() == 0 {
				return
			}
		}
	}
}
