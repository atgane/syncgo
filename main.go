package main

import (
	"errors"
	"fmt"
)

func main() {
	fmt.Println(example())
}

func example() (result int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("recovered from panic")
		}
	}()

	// 패닉이 발생할 코드
	panic("something went wrong")

	return 0, nil // 이 줄은 도달하지 않습니다
}
