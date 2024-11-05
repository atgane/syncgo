package main

import "fmt"

func main() {
	q := make(chan struct{}, 10)
	q <- struct{}{}
	q <- struct{}{}
	q <- struct{}{}
	q <- struct{}{}

	close(q)
	fmt.Println(len(q))
}
