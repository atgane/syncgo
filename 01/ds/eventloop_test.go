package ds

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestEventloopGoroutineLeak(t *testing.T) {
	defer goleak.VerifyNone(t)

	handler := func(event int) {}
	el := NewEventloop(1024, 2, handler)

	go el.Run()

	time.Sleep(time.Millisecond)
	el.Close()
	time.Sleep(time.Millisecond)
}

func TestEventloopRace(t *testing.T) {
	handler := func(event int) {
		time.Sleep(time.Millisecond)
	}
	el := NewEventloop(1024, 2, handler)

	go el.Run()
	time.Sleep(time.Millisecond)

	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := el.Send(i)
			require.NoError(t, err)
		}(i)
	}
	wg.Wait()
	el.Close()
}

func TestEventloopChannelClose(t *testing.T) {
	handler := func(event int) {}
	el := NewEventloop(1024, 2, handler)

	go el.Run()
	time.Sleep(time.Millisecond)

	err := el.Send(1)
	require.NoError(t, err)

	el.Close()

	err = el.Send(2)
	require.ErrorIs(t, err, ErrAlreadyClosedLoop)
}

func TestEventloopCloseSyncEnd(t *testing.T) {
	var mu sync.Mutex
	processed := make(map[int]bool)

	handler := func(event int) {
		mu.Lock()
		defer mu.Unlock()
		processed[event] = true
	}

	el := NewEventloop(1024, 2, handler)

	go el.Run()
	time.Sleep(time.Millisecond)

	for i := range 10 {
		err := el.Send(i)
		require.NoError(t, err)
	}

	el.Close()
	time.Sleep(time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, processed, 10)
}

func TestEventloopCloseSyncRunning(t *testing.T) {
	var mu sync.Mutex
	processed := make(map[int]bool)

	handler := func(event int) {
		mu.Lock()
		defer mu.Unlock()
		processed[event] = true
	}

	el := NewEventloop(1024, 2, handler)

	go el.Run()
	time.Sleep(time.Millisecond)

	var wg sync.WaitGroup

	n := 100
	sendCount := atomic.Int64{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := el.Send(i); err == nil {
				sendCount.Add(1)
			}
		}(i)
	}

	wg.Add(1)
	go el.Close()
	wg.Done()
	wg.Wait()

	// 정상 종료 시 Send에서 에러가 없던 이벤트는 정상 실행되어야 함
	require.Equal(t, int(sendCount.Load()), len(processed))
}

func TestEventloopForceCloseSyncRunning(t *testing.T) {
	handler := func(event int) {
		time.Sleep(time.Second)
	}

	el := NewEventloop(1024, 2, handler)

	go el.Run()
	time.Sleep(time.Millisecond)

	for i := 0; i < 10; i++ {
		el.Send(i)
	}

	el.ForceClose()

	// 강제 종료 시 queue에 원소가 남아있는 상태
	require.Less(t, 0, len(el.queue))
}
