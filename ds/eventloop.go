package ds

import (
	"context"
	"errors"
	"sync"
)

type Eventloop[T any] struct {
	q  chan T
	h  Handler[T]
	c  *Closer
	dc int
}

type Handler[T any] func(T)

type EventloopConfig struct {
	QueueSize       int
	DispatcherCount int
}

func NewEventloopConfig() *EventloopConfig {
	return &EventloopConfig{
		QueueSize:       1024,
		DispatcherCount: 1,
	}
}

func NewEventloop[T any](handler Handler[T], config *EventloopConfig) *Eventloop[T] {
	e := &Eventloop[T]{
		q:  make(chan T, config.QueueSize),
		h:  handler,
		c:  NewCloser(),
		dc: config.DispatcherCount,
	}

	return e
}

func (e *Eventloop[T]) Run() {
	// 정해진 개수만큼 고루틴을 미리 생성
	wg := sync.WaitGroup{}
	for range e.dc {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.dispatch()
		}()
	}

	e.c.Wait()
	wg.Wait()
}

func (e *Eventloop[T]) dispatch() {
	for {
		select {
		case t, ok := <-e.q: // 이벤트 우선 처리
			if !ok {
				return
			}
			e.h(t)
			continue
		default:
		}

		select {
		case t, ok := <-e.q:
			if !ok {
				return
			}
			e.h(t)
			continue
		case <-e.c.Done():
			return
		}
	}
}

func (e *Eventloop[T]) Send(ctx context.Context, event T) error {
	select {
	case <-e.c.Done(): // 닫힘 우선 처리
		return errors.New("already closed eventloop")
	default:
	}

	select {
	case <-e.c.Done():
		return errors.New("already closed eventloop")
	case <-ctx.Done():
		return ctx.Err()
	case e.q <- event:
		return nil
	}
}

func (e *Eventloop[T]) Close() {
	e.c.Close()
}

func (e *Eventloop[T]) ForceClose() {
	e.c.Close()
	close(e.q)
}
