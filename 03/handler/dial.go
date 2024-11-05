package handler

import "net"

func DialConn[T any](
	address string,
	handler TCPServerHandler[T],
	config *ConnConfig,
) (*Conn[T], error) {
	c, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	conn := newConn[T](c, handler, config)
	go func() {
		handler.OnOpen(conn)
		conn.run()
		handler.OnClose(conn)
	}()

	return conn, nil
}
