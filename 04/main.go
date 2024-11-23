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

	// 6. epoll fd 생성
	epollfd, err := unix.EpollCreate1(0)
	if err != nil {
		fmt.Println("epollcreate error", err)
		return
	}

	go func() {
		for {
			// 7. accept를 호출하여 클라이언트 연결 대기
			// Accept4호출 이유는 클라이언트측 소켓에 옵션을 부여하여 생성할 수 있기 때문
			// nfd, _, err := unix.Accept4(fd, syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC)
			nfd, _, err := unix.Accept4(fd, unix.SOCK_CLOEXEC)
			if err != nil {
				fmt.Println("accept4 error", err)
				continue
			}

			// 8. accept가 성공하면 epoll fd에 이벤트 등록
			// accept된 fd에 unix.EPOLLIN | unix.EPOLLHUP | unix.EPOLLERR에 대한 이벤트 트리거
			if err := unix.EpollCtl(epollfd, unix.EPOLL_CTL_ADD, nfd, &unix.EpollEvent{
				Fd:     int32(nfd),
				Events: unix.EPOLLIN | unix.EPOLLHUP | unix.EPOLLERR,
			}); err != nil {
				fmt.Println("epolladd error", err)
				continue
			}
		}
	}()

	go func() {
		events := make([]unix.EpollEvent, 128)
		for {
			// 9. epoll fd에서 발생하는 이벤트 대기
			n, err := unix.EpollWait(epollfd, events, -1)
			if err != nil {
				fmt.Println("epollwait error", err)
				continue
			}

			// 10. event 상태 체크
			b := make([]byte, 1024)
			for i := range n {
				event := events[i]
				var errfd error
				// 11. 일반 소켓에 대한 입력이 트리거된 경우 일반 로직 진행
				if event.Events&unix.EPOLLIN == unix.EPOLLIN {
					errfd = logic(int(event.Fd), b)
				}

				// 12. 소켓에에러가 발생한 경우 에러로 취급
				if event.Events&(unix.EPOLLERR|unix.EPOLLHUP) != 0 {
					errfd = fmt.Errorf("epoll hup error")
				}

				// 13. epoll fd에서 소켓 제거
				if errfd != nil {
					fmt.Println("evnet fd error", errfd)
					unix.EpollCtl(epollfd, unix.EPOLL_CTL_DEL, int(event.Fd), &unix.EpollEvent{
						Fd:     int32(event.Fd),
						Events: 0,
					})
				}
			}
		}
	}()

	select {}
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
