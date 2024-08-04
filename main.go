package main

import "fmt"

func main() {
	c := make(chan struct{})
	go func() {
		close(c)
	}()
	_, ok := <-c
	fmt.Println(ok)
}
