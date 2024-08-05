package ds

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type Eventloop[T any] struct {
	q       chan T
	h       Handler[T]
	dc      int          // dispatch 개수
	sender  atomic.Int32 // Send중인 컨텍스트 개수
	closeCh chan struct{}
	closed  atomic.Bool
}

type Handler[T any] func(T)

type EventloopConfig struct {
	QueueSize       int
	DispatcherCount int
}

// EventloopConfig 생성. 기본 설정에서 하나의 고루틴으로 Dispatcher 처리
func NewEventloopConfig() *EventloopConfig {
	return &EventloopConfig{
		QueueSize:       1024,
		DispatcherCount: 1,
	}
}

// Eventloop 생성
func NewEventloop[T any](handler Handler[T], config *EventloopConfig) *Eventloop[T] {
	e := &Eventloop[T]{
		q:       make(chan T, config.QueueSize),
		h:       handler,
		dc:      config.DispatcherCount,
		closeCh: make(chan struct{}),
	}

	return e
}

// 이벤트루프 블로킹 실행.
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

	wg.Wait()
}

// 이벤트 전달 메서드. 외부 ctx 종료, Eventloop 닫힘에 대하여 에러 리턴.
// Send가 정상 처리된 경우 Close가 호출되어도 실행 보장.
func (e *Eventloop[T]) Send(ctx context.Context, event T) error {
	e.sender.Add(1)
	defer e.sender.Add(-1)

	if e.closed.Load() {
		return ErrorAlreadyClose
	}

	// context 종료 우선 처리
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	select {
	case <-e.closeCh:
		return ErrorAlreadyClose
	case <-ctx.Done():
		return ctx.Err()
	case e.q <- event:
		// Close 호출 이후에 event가 들어가는 경우 dispatch 스핀
		return nil
	}
}

// 정상 종료. 추가 이벤트를 방지하고 현재 존재하는 이벤트를 모두 실행 후 종료
func (e *Eventloop[T]) Close() {
	if !e.closed.CompareAndSwap(false, true) {
		return
	}
	close(e.closeCh)
}

// 강제 종료. 추가 이벤트 방지 및 존재하는 이벤트 무시하고 종료
func (e *Eventloop[T]) ForceClose() {
	if !e.closed.CompareAndSwap(false, true) {
		return
	}
	close(e.closeCh) // Send() wake
	close(e.q)
}

var ErrorAlreadyClose = errors.New("already closed channel")

func (e *Eventloop[T]) dispatch() {
	for {
		select {
		case t, ok := <-e.q:
			if !ok {
				return
			}
			e.h(t)
		case <-e.closeCh:
			if e.sender.Load() == 0 && len(e.q) == 0 {
				// 모든 Send() 종료 확인 & 이벤트 없음 확인
				return
			}
			// 이 외의 경우 조건부 스핀
		}
	}
}
