package ds

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapWork(t *testing.T) {
	m := NewMap[int, int](5)

	m.Store(1, 1000)
	require.Equal(t, 1000, m.m[1])

	var v int
	var ok bool
	v, ok = m.Load(1)
	require.Equal(t, 1000, v)
	require.Equal(t, true, ok)

	v, ok = m.Load(0)
	require.Equal(t, 0, v)
	require.Equal(t, false, ok)

	m.Delete(1)
	require.Equal(t, 0, len(m.m))

	m.Store(2, 2000)
	old, loaded := m.Swap(2, 2001)
	require.Equal(t, 2000, old)
	require.Equal(t, true, loaded)
	old, loaded = m.Swap(1, 1000)
	require.Equal(t, 0, old)
	require.Equal(t, false, loaded)

	v, loaded = m.LoadAndDelete(1)
	require.Equal(t, 1000, v)
	require.Equal(t, true, loaded)
	v, loaded = m.LoadAndDelete(0)
	require.Equal(t, 0, v)
	require.Equal(t, false, loaded)

	p, loaded := m.LoadOrStore(2, 2000)
	require.Equal(t, 2000, p)
	require.Equal(t, true, loaded)
	p, loaded = m.LoadOrStore(1, 1000)
	require.Equal(t, 1000, p)
	require.Equal(t, false, loaded)

	keys := make(map[int]int)
	m.Range(func(k int, v int) bool {
		keys[k] = v
		return true
	})
	require.Equal(t, 1000, keys[1])
	require.Equal(t, 2000, keys[2])
}

func TestMapRangeDeadLock(t *testing.T) {
	m := NewMap[int, int](8)
	m.Store(1, 101)
	m.Store(2, 202)
	m.Store(3, 303)
	m.Store(4, 404)
	m.Store(5, 505)
	m.Store(6, 606)
	m.Store(7, 707)
	m.Store(8, 808)

	m.Range(func(k, v int) bool {
		if k == 8 {
			m.Delete(k)
		}
		return true
	})
}
