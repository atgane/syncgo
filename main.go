package main

import (
	"fmt"
	"syncgo/ds"
)

func main() {
	m := ds.NewMap[int, int](8)
	m.Store(1, 101)
	m.Store(2, 202)
	m.Store(3, 303)
	m.Store(4, 404)
	m.Store(5, 505)
	m.Store(6, 606)
	m.Store(7, 707)
	m.Store(8, 808)

	fmt.Println(m.LoadOrStore(1, 100))

	m.Range(func(key, value int) bool {
		if key == 2 {
			m.Delete(2)
			m.Store(20, 2020)
		}
		fmt.Println(key, value)
		return true
	})
}
