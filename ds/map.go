package ds

import "sync"

type Map[F comparable, T any] struct {
	mu sync.RWMutex
	m  map[F]T
}

func NewMap[F comparable, T any](initSize int) *Map[F, T] {
	r := new(Map[F, T])
	r.m = make(map[F]T, initSize)
	return r
}

func (m *Map[F, T]) Load(k F) (v T, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok = m.m[k]
	return v, ok
}

func (m *Map[F, T]) Store(k F, v T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.m[k] = v
}

func (m *Map[F, T]) Delete(k F) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.m, k)
}

func (m *Map[F, T]) Swap(k F, v T) (p T, loaded bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, loaded = m.m[k]
	m.m[k] = v
	return p, loaded
}

func (m *Map[F, T]) LoadAndDelete(k F) (v T, loaded bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, loaded = m.m[k]
	delete(m.m, k)
	return v, loaded
}

func (m *Map[F, T]) LoadOrStore(k F, v T) (T, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, loaded := m.m[k]
	m.m[k] = v
	if !loaded {
		return v, false
	}
	return v, true
}

func (m *Map[F, T]) Range(f func(F, T) bool) {
	m.mu.RLock()
	length := len(m.m)
	ks := make([]F, 0, length)
	vs := make([]T, 0, length)
	for k, v := range m.m {
		ks = append(ks, k)
		vs = append(vs, v)
	}
	m.mu.RUnlock()
	for i := range length {
		if !f(ks[i], vs[i]) {
			break
		}
	}
}
