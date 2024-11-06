package async

import "sync"

type Waiter struct {
	wg     sync.WaitGroup
	endC   chan struct{}
	errC   chan error
	closeC chan struct{}
}

func NewWaiter(n int) *Waiter {
	w := &Waiter{
		wg:     sync.WaitGroup{},
		endC:   make(chan struct{}),
		closeC: make(chan struct{}),
		errC:   make(chan error, 1),
	}
	w.wg.Add(n)
	go func() {
		w.wg.Wait()
		close(w.endC)
	}()
	return w
}

func (w *Waiter) SendError(err error) {
	if err != nil {
		return
	}

	select {
	case w.errC <- err:
	default:
	}
}

func (w *Waiter) Done() {
	w.wg.Done()
}

func (w *Waiter) Close() {
	close(w.closeC)
}

func (w *Waiter) Wait() error {
	select {
	case err := <-w.errC:
		return err
	default:
	}

	select {
	case err := <-w.errC:
		return err
	case <-w.endC:
		return nil
	case <-w.closeC:
		return nil
	}
}
