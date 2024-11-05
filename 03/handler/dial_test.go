package handler

import "syncgo/test/async"

type testClient1 struct {
	errWaiter *async.Waiter
	server    *TCPServer[struct{}]
}

func newTestClient1(port int, errWaiter *async.Waiter) *testClient1 {
	serverConfig := NewTCPServerConfig()
	serverConfig.Port = port

	h := &testClient1{}
	h.errWaiter = errWaiter
	h.server = NewTCPServer(h, serverConfig)
	return h
}

func (h *testClient1) OnOpen(conn *Conn[struct{}])                  {}
func (h *testClient1) OnClose(conn *Conn[struct{}])                 {}
func (h *testClient1) OnReadError(conn *Conn[struct{}], err error)  {}
func (h *testClient1) OnWriteError(conn *Conn[struct{}], err error) {}

func (h *testClient1) OnRead(conn *Conn[struct{}], b []byte) int {
	p, n := unmarshal(b)
	if n == 0 {
		return 0
	}
	d, err := marshal(p)
	if err != nil {
		h.errWaiter.SendError(err)
		return n
	}
	h.server.ConnMap.Range(func(i int, c *Conn[struct{}]) bool {
		if err := c.Write(d); err != nil {
			h.errWaiter.SendError(err)
			return false
		}
		return true
	})
	return n
}
