package main

import (
	"fmt"
)

func dealWithErr(err1, err2 error) (res error) {
	defer func() {
		res = err1
	}()
	return err2
}

func dealWithErr2(err1, err2 error) (res error) {
	defer func() {
		res = err1
	}()
	res = err2
	return
}

func dealWithErr3(err1, err2 error) (res error) {
	defer func() {
		if res != nil {
			res = fmt.Errorf("%v: %w", err1, res)
		} else {
			res = err1
		}
	}()
	// res = err2
	return err2
}

func main() {
	fmt.Println("dealWithErr:")
	fmt.Println("nil, nil:", dealWithErr(nil, nil))
	fmt.Println("nil, err2:", dealWithErr(nil, fmt.Errorf("err2")))
	fmt.Println("err1, nil:", dealWithErr(fmt.Errorf("err1"), nil))
	fmt.Println("err1, err2:", dealWithErr(fmt.Errorf("err1"), fmt.Errorf("err2")))

	fmt.Println("dealWithErr2:")
	fmt.Println("nil, nil:", dealWithErr2(nil, nil))
	fmt.Println("nil, err2:", dealWithErr2(nil, fmt.Errorf("err2")))
	fmt.Println("err1, nil:", dealWithErr2(fmt.Errorf("err1"), nil))
	fmt.Println("err1, err2:", dealWithErr2(fmt.Errorf("err1"), fmt.Errorf("err2")))

	fmt.Println("dealWithErr3:")
	fmt.Println("nil, nil:", dealWithErr3(nil, nil))
	fmt.Println("nil, err2:", dealWithErr3(nil, fmt.Errorf("err2")))
	fmt.Println("err1, nil:", dealWithErr3(fmt.Errorf("err1"), nil))
	fmt.Println("err1, err2:", dealWithErr3(fmt.Errorf("err1"), fmt.Errorf("err2")))

}
