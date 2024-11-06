package handler

import (
	"fmt"
	"syncgo/ds"
	"syncgo/test/async"

	"github.com/rs/zerolog/log"
)

type testClient1 struct {
	port        int
	message     []byte
	readWaiter  *async.Waiter
	writeWaiter *async.Waiter
	closeWaiter *async.Waiter
	conn        *Conn[int]

	offset int
	data   *ds.Map[int, *testPacket]
}

func newTestClient1(
	port int,
	message []byte,
	readWaiter,
	writeWaiter,
	closeWaiter *async.Waiter,
) *testClient1 {
	c := &testClient1{}
	c.port = port
	c.message = message
	c.readWaiter = readWaiter
	c.writeWaiter = writeWaiter
	c.closeWaiter = closeWaiter

	c.data = ds.NewMap[int, *testPacket](100)
	return c
}

func (h *testClient1) Run() error {
	var err error
	h.conn, err = DialConn(fmt.Sprintf(":%d", h.port), h, newConnConfig())
	return err
}

func (h *testClient1) WritePacket() error {
	testPacket := newTestPacket(0, uint64(h.conn.Id), uint32(h.offset), h.message)
	b, err := marshal(testPacket)
	if err != nil {
		log.Error().
			Err(err).
			Uint64("from_session", uint64(h.conn.Id)).
			Uint32("offset", uint32(h.offset)).
			Msg("client write packet error")
		return err
	}
	log.Info().
		Uint64("from_session", uint64(h.conn.Id)).
		Uint32("offset", uint32(h.offset)).
		Msg("client write packet")
	h.offset++
	return h.conn.Write(b)
}

func (h *testClient1) OnOpen(conn *Conn[int])                  { conn.Field = conn.Id }
func (h *testClient1) OnClose(conn *Conn[int])                 { h.closeWaiter.Done() }
func (h *testClient1) OnReadError(conn *Conn[int], err error)  { h.readWaiter.SendError(err) }
func (h *testClient1) OnWriteError(conn *Conn[int], err error) { h.writeWaiter.SendError(err) }

func (h *testClient1) OnRead(conn *Conn[int], b []byte) int {
	p, n := unmarshal(b)
	if n == 0 {
		return 0
	}

	defer h.readWaiter.Done()
	sessionId := int(p.Header.Session)
	prevp, ok := h.data.Load(sessionId)
	if !ok {
		if p.Header.Offset != 0 {
			h.readWaiter.SendError(fmt.Errorf("session %d invalid offset %d", sessionId, p.Header.Offset))
			return n
		}
		h.data.Store(sessionId, p)
		return n
	}

	if p.Header.Offset != prevp.Header.Offset+1 {
		h.readWaiter.SendError(fmt.Errorf("session %d invalid offset %d", sessionId, prevp.Header.Offset))
		return n
	}

	if p.Header.Status != 0 {
		h.readWaiter.SendError(fmt.Errorf("session %d wrong status %d", sessionId, p.Header.Status))
		return n
	}

	if len(p.Body) != int(p.Header.BodySize) {
		h.readWaiter.SendError(fmt.Errorf("session %d invalid body size %d, body %v", sessionId, p.Header.BodySize, p.Body))
		return n
	}

	if string(p.Body) != string(h.message) {
		h.readWaiter.SendError(fmt.Errorf("session %d invalid message %v", sessionId, p.Body))
		return n
	}

	log.Info().
		Uint64("from_session", p.Header.Session).
		Uint64("to_session", uint64(h.conn.Id)).
		Uint32("offset", p.Header.Offset).
		Msg("client read packet")

	h.data.Store(sessionId, p)
	return n
}
