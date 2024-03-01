package main

import (
	"fmt"
	"time"
)

// 6. What is a goroutine? How do you stop it?
func routine(ch chan int, quit chan struct{}) {
	defer fmt.Println("Leaving routine...")
	for {
		select {
		case v := <-ch:
			fmt.Println("Got", v)
		case <-quit:
			return
		}
	}
}

// 7. How do you check a variable type at runtime?
func checkType(v interface{}) {
	switch v.(type) {
	case int, int16, int32, int64:
		fmt.Println("Integer")
	case string:
		fmt.Println("String")
	}
}

// 10. What are function closures?
func closure(a string, f func(string) string) {
	fmt.Printf("%s - %s - %s\n", a, f("ric"), a)
}
func closure2(a int) func(int) int {
	return func(i int) int {
		return a + i
	}
}

func main() {
	fmt.Println("start")
	ch := make(chan int)
	quit := make(chan struct{})
	go routine(ch, quit)
	ch <- 1
	ch <- 2
	quit <- struct{}{}
	time.Sleep(100 * time.Millisecond)
	//
	checkType(12)
	checkType("Ricardo")
	//
	f := func(a string) string {
		return "!" + a + "!"
	}
	closure("hello", f)
	// closure2
	add2 := closure2(2)
	fmt.Println("add2(3) =", add2(3))
}
