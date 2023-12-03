package main

import (
	"errors"
	"fmt"
)

type MyError1 struct {
	Msg string
}

func (e *MyError1) Error() string {
	return e.Msg
}

type MyError2 struct {
	Msg string
}

func (e *MyError2) Error() string {
	return e.Msg
}

func main() {
	err1 := &MyError1{"err1"}

	err1Copy := err1

	if errors.Is(err1Copy, err1) {
		fmt.Println("err1Copy is err1")
	}

	var e1 *MyError1
	if errors.As(err1Copy, &e1) {
		fmt.Println("err1Copy is *MyError:", e1)
	}

	e11 := &MyError1{}
	if errors.As(err1Copy, &e11) {
		fmt.Println("err1Copy is *MyError:", e11)
	}

	var e2 *MyError2
	if errors.As(err1Copy, &e2) {
		fmt.Println("err1Copy is *MyError2:", e2)
	} else {
		fmt.Println("err1Copy is not *MyError2", e2)
	}

}
