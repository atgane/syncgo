package handler

import (
	"net"
	"sync"
	"sync/atomic"
	"syncgo/ds"
	"time"
)

type Conn[T any] struct {
	Id             int
	Field          T
	conn           net.Conn
	handler        TCPServerHandler[T]
	config         *ConnConfig
	closed         atomic.Bool
	writeEventloop *ds.Eventloop[[]byte]
}

func newConnConfig() *ConnConfig {
	return &ConnConfig{
		BufferSize:         4096,
		ReadTimeoutSecond:  600,
		WriteTimeoutSecond: 600,
	}
}

type ConnConfig struct {
	BufferSize         int
	ReadTimeoutSecond  int
	WriteTimeoutSecond int
}

func newConn[T any](
	conn net.Conn,
	handler TCPServerHandler[T],
	config *ConnConfig,
) *Conn[T] {
	c := &Conn[T]{
		Id:      int(time.Now().UnixNano()),
		conn:    conn,
		handler: handler,
		config:  config,
	}
	c.writeEventloop = ds.NewEventloop(1, 16, c.onWrite)
	return c
}

func (c *Conn[T]) Write(b []byte) error {
	return c.writeEventloop.Send(b)
}

func (c *Conn[T]) Close() {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}

	c.writeEventloop.Close()
	c.conn.Close()
}

func (c *Conn[T]) run() {
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		c.onRead()
	}()
	go func() {
		defer wg.Done()
		c.writeEventloop.Run()
	}()

	wg.Wait()
}

func (c *Conn[T]) onRead() {
	defer c.Close()

	b := make([]byte, c.config.BufferSize)
	buf := make([]byte, 0, c.config.BufferSize)
	for {
		n, err := c.conn.Read(b)
		if err != nil {
			if c.closed.Load() {
				return
			}
			c.handler.OnReadError(c, err)
			return
		}

		buf = append(buf, b[:n]...)
		p := 0
		read := 0
		l := len(buf)
		for read < l {
			p = c.handler.OnRead(c, buf[read:])
			if p == 0 {
				break
			}
			read += p
		}
		buf = buf[read:]
	}
}

func (c *Conn[T]) onWrite(b []byte) {
	var err error
	p := 0
	write := 0
	l := len(b)
	for write < l {
		p, err = c.conn.Write(b[write:])
		if err != nil {
			if c.closed.Load() {
				return
			}
			c.handler.OnWriteError(c, err)
			return
		}
		write += p
	}
}
