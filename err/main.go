package main

import "fmt"

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

func main() {
	fmt.Println("dealWithErr:")
	fmt.Println("nil, nil:", dealWithErr(nil, nil))
	fmt.Println("nil, err:", dealWithErr(nil, fmt.Errorf("err")))
	fmt.Println("err, nil:", dealWithErr(fmt.Errorf("err"), nil))
	fmt.Println("err, err:", dealWithErr(fmt.Errorf("err1"), fmt.Errorf("err2")))

	fmt.Println("dealWithErr2:")
	fmt.Println("nil, nil:", dealWithErr2(nil, nil))
	fmt.Println("nil, err:", dealWithErr2(nil, fmt.Errorf("err")))
	fmt.Println("err, nil:", dealWithErr2(fmt.Errorf("err"), nil))
	fmt.Println("err, err:", dealWithErr2(fmt.Errorf("err1"), fmt.Errorf("err2")))
}
