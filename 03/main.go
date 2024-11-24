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

	// 2. setsopckopt으로 fd에 소켓 옵션 지정
	// 여기서는 REUSEADDR 플래그 활성화
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		fmt.Println("set sock opt error", err)
		return
	}

	// // 3. 서버측 fd에 대한 blocking 설정
	// if err := unix.SetNonblock(fd, true); err != nil {
	// 	fmt.Println("set non-block error", err)
	// 	return
	// }

	// 4. ipv4기반 주소 생성 및 서버측 소켓과 주소 바인딩
	addr := &unix.SockaddrInet4{}
	addr.Addr = [4]byte{0, 0, 0, 0}
	addr.Port = 4999
	if err := unix.Bind(fd, addr); err != nil {
		fmt.Println("bind error", err)
		return
	}

	// 5. backlog queue 사이즈 설정 및 서버 호스팅
	queueSize := 1024
	if err := unix.Listen(fd, queueSize); err != nil {
		fmt.Println("listen error", err)
		return
	}

	for {
		// 6. accept를 호출하여 클라이언트 연결 대기
		// Accept4호출 이유는 클라이언트측 소켓에 옵션을 부여하여 생성할 수 있기 때문
		// nfd, _, err := unix.Accept4(fd, syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC)
		nfd, _, err := unix.Accept4(fd, unix.SOCK_CLOEXEC)
		if err != nil {
			fmt.Println("accept4 error", err)
			continue
		}

		// 7. accept의 리턴으로 받은 클라이언트측 소켓에 대한 file discriptor를 통해
		// r/w를 진행
		go func(nfd int) {
			b := make([]byte, 1024)
			for {
				if err := logic(nfd, b); err != nil {
					fmt.Println("fd error", err)
					unix.Close(int(nfd))
					break
				}
			}
		}(nfd)
	}
}

func logic(fd int, b []byte) error {
	r, err := unix.Read(fd, b)
	if err != nil {
		return err
	}

	if r == 0 {
		return fmt.Errorf("read EOF")
	}

	_, err = unix.Write(fd, b[:r])
	if err != nil {
		return err
	}
	return nil
}
