package main

import (
	"errors"
	"fmt"
)

// it prints "err defer"
func f() (err error) {
	defer func() {
		err = errors.New("err defer")
	}()
	return errors.New("err return")
}

func main() {

	fmt.Println(f())
}
