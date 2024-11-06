package handler

import (
	"fmt"
	"net"
	"syncgo/test/async"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
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
	acceptWaiter *async.Waiter
	errWaiter    *async.Waiter
	server       *TCPServer[int]
}

func newTestServer1(port int, acceptWaiter *async.Waiter, errWaiter *async.Waiter) *testServer1 {
	serverConfig := NewTCPServerConfig()
	serverConfig.Port = port

	h := &testServer1{}
	h.acceptWaiter = acceptWaiter
	h.errWaiter = errWaiter
	h.server = NewTCPServer(h, serverConfig)
	return h
}

func (h *testServer1) Run() error {
	return h.server.Run()
}

func (h *testServer1) OnOpen(conn *Conn[int])                  { h.acceptWaiter.Done() }
func (h *testServer1) OnClose(conn *Conn[int])                 {}
func (h *testServer1) OnReadError(conn *Conn[int], err error)  { h.errWaiter.SendError(err) }
func (h *testServer1) OnWriteError(conn *Conn[int], err error) { h.errWaiter.SendError(err) }

func (h *testServer1) OnRead(conn *Conn[int], b []byte) int {
	p, n := unmarshal(b)
	if n == 0 {
		return 0
	}
	log.Info().
		Uint64("from_session", p.Header.Session).
		Uint32("offset", p.Header.Offset).
		Msg("server read packet")
	conn.Field = int(p.Header.Session)
	d, err := marshal(p)
	if err != nil {
		h.errWaiter.SendError(err)
		return n
	}
	h.server.ConnMap.Range(func(i int, c *Conn[int]) bool {
		if err := c.Write(d); err != nil {
			log.Error().
				Err(err).
				Uint64("from_session", p.Header.Session).
				Uint64("to_session", uint64(c.Field)).
				Uint32("offset", p.Header.Offset).
				Msg("server write packet error")
			h.errWaiter.SendError(err)
			return false
		}
		log.Info().
			Uint64("from_session", p.Header.Session).
			Uint64("to_session", uint64(c.Field)).
			Uint32("offset", p.Header.Offset).
			Msg("server write packet")
		return true
	})
	return n
}

func TestTcpServerClient1(t *testing.T) {
	port := findport()
	clientCount := 20
	messageCount := 20
	message := []byte("hello world")
	runWaiter := async.NewWaiter(1)
	errWaiter := async.NewWaiter(1)
	acceptWaiter := async.NewWaiter(clientCount + 1)
	readWaiter := async.NewWaiter(clientCount * clientCount * messageCount)
	writeWaiter := async.NewWaiter(clientCount * messageCount)
	closeWaiter := async.NewWaiter(clientCount)

	s := newTestServer1(port, acceptWaiter, errWaiter)
	go func() {
		errWaiter.SendError(s.Run())
	}()

	go func() {
		for {
			_, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
			if err == nil {
				runWaiter.Done()
				return
			}
		}
	}()

	require.NoError(t, runWaiter.Wait())
	clients := []*testClient1{}

	for range clientCount {
		clients = append(clients, newTestClient1(port, message, readWaiter, writeWaiter, closeWaiter))
	}

	for _, client := range clients {
		require.NoError(t, client.Run())
	}

	require.NoError(t, acceptWaiter.Wait())

	for _, client := range clients {
		go func() {
			for range messageCount {
				writeWaiter.SendError(client.WritePacket())
			}
		}()
	}

	require.NoError(t, writeWaiter.Wait())
	time.Sleep(time.Second)
	for range 2 {
		fmt.Printf("=========================================\n")
		for idx, client := range clients {
			fmt.Printf("-------------------------------\n")
			fmt.Printf("index: %d\n", idx)
			fmt.Printf("client id: %d\n", client.conn.Id)
			fmt.Printf("write offset: %d\n", client.offset)
			sum := 0
			client.data.Range(func(key int, value *testPacket) bool {
				fmt.Printf("read client id: %d, offset: %d\n", key, value.Header.Offset)
				sum += int(value.Header.Offset)
				return true
			})
			fmt.Printf("read offset: %d\n", sum)
		}
		time.Sleep(time.Second)
	}
	require.NoError(t, readWaiter.Wait())
	require.NoError(t, errWaiter.Wait())
}
