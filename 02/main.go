package main

import (
	"fmt"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":4999")
	if err != nil {
		fmt.Println("listen error", err)
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("accept error", err)
			continue
		}

		go func(conn net.Conn) {
			b := make([]byte, 1024)
			for {
				r, err := conn.Read(b)
				if err != nil {
					fmt.Println("read error", err)
					break
				}

				_, err = conn.Write(b[:r])
				if err != nil {
					fmt.Println("write error", err)
					break
				}
			}
		}(conn)
	}
}
