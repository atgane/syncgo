package main

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func main() {
	// 1. 서버측 소켓 생성 후 소켓에 대한 file discriptor 소유
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		fmt.Println("socket error", err)
		return
	}

	// 2. 서버측 fd에 대한 blocking 설정
	if err := unix.SetNonblock(fd, true); err != nil {
		fmt.Println("set non-block error", err)
		return
	}

	// 3. ipv4기반 주소 생성 및 서버측 소켓과 주소 바인딩
	addr := &unix.SockaddrInet4{}
	addr.Addr = [4]byte{0, 0, 0, 0}
	addr.Port = 4999
	if err := unix.Bind(fd, addr); err != nil {
		fmt.Println("bind error", err)
		return
	}

	// 4. backlog queue 사이즈 설정 및 서버 호스팅
	queueSize := 1024
	if err := unix.Listen(fd, queueSize); err != nil {
		fmt.Println("listen error", err)
		return
	}

	for {
		// 5. accept를 호출하여 클라이언트 연결 대기
		// Accept4호출 이유는 클라이언트측 소켓에 옵션을 부여하여 생성할 수 있기 때문
		// nfd, _, err := unix.Accept4(fd, syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC)
		nfd, _, err := unix.Accept4(fd, unix.SOCK_CLOEXEC)
		if err != nil {
			fmt.Println("accept4 error", err)
			continue
		}

		// 6. accept의 리턴으로 받은 클라이언트측 소켓에 대한 file discriptor를 통해
		// r/w를 진행
		go func(nfd int) {
			b := make([]byte, 1024)
			for {
				r, err := unix.Read(nfd, b)
				if err != nil {
					fmt.Println("read error", err)
					break
				}

				if r == 0 {
					fmt.Println("read EOF")
					break
				}

				_, err = unix.Write(nfd, b[:r])
				if err != nil {
					fmt.Println("write error", err)
					break
				}
			}
		}(nfd)
	}
}
