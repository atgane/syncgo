package handler

import (
	"fmt"
	"net"
	"sync/atomic"
	"syncgo/ds"
)

type TCPServer[T any] struct {
	ConnMap  *ds.Map[int, *Conn[T]]
	listener net.Listener
	handler  TCPServerHandler[T]
	config   *TCPServerConfig
	closed   atomic.Bool
}

type TCPServerConfig struct {
	*ConnConfig
	Port int
}

func NewTCPServerConfig() *TCPServerConfig {
	return &TCPServerConfig{
		ConnConfig: newConnConfig(),
	}
}

type TCPServerHandler[T any] interface {
	OnOpen(conn *Conn[T])
	OnClose(conn *Conn[T])
	OnRead(conn *Conn[T], b []byte) (n int)
	OnReadError(conn *Conn[T], err error)
	OnWriteError(conn *Conn[T], err error)
}

func NewTCPServer[T any](handler TCPServerHandler[T], config *TCPServerConfig) *TCPServer[T] {
	s := &TCPServer[T]{
		ConnMap: ds.NewMap[int, *Conn[T]](1024),
		handler: handler,
		config:  config,
	}
	return s
}

func (s *TCPServer[T]) Run() error {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
	if err != nil {
		return err
	}

	for {
		c, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return nil
			}
			return err
		}

		go func(c net.Conn) {
			conn := newConn[T](c, s.handler, s.config.ConnConfig)

			s.ConnMap.Store(conn.Id, conn)
			s.handler.OnOpen(conn)
			conn.run()
			s.handler.OnClose(conn)
			s.ConnMap.Delete(conn.Id)
		}(c)
	}
}

func (s *TCPServer[T]) Close() {
	if !s.closed.CompareAndSwap(false, true) {
		return
	}

	s.listener.Close()
}
