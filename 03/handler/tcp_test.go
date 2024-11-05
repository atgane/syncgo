package handler

import (
	"net"
	"syncgo/test/async"
	"testing"
)

func findport() int {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	return port
}

type testServer1 struct {
	errWaiter *async.Waiter
	server    *TCPServer[struct{}]
}

func newTestServer1(port int, errWaiter *async.Waiter) *testServer1 {
	serverConfig := NewTCPServerConfig()
	serverConfig.Port = port

	h := &testServer1{}
	h.errWaiter = errWaiter
	h.server = NewTCPServer(h, serverConfig)
	return h
}

func (h *testServer1) OnOpen(conn *Conn[struct{}])                  {}
func (h *testServer1) OnClose(conn *Conn[struct{}])                 {}
func (h *testServer1) OnReadError(conn *Conn[struct{}], err error)  {}
func (h *testServer1) OnWriteError(conn *Conn[struct{}], err error) {}

func (h *testServer1) OnRead(conn *Conn[struct{}], b []byte) int {
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

func TestTcpServerClient1(t *testing.T) {

}
