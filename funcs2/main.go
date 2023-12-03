package main

import (
	"fmt"
)

// cal labeler with a function signature adapter

func convertToBin(n int) string {
	result := ""
	for ; n > 0; n /= 2 {
		lsb := n % 2
		result = fmt.Sprintf("%d%s", lsb, result) // Sprintf格式化输出
	}
	return result
}

func convert2(f float64) func(int) string {
	f = 1.0 / f
	return func(n int) string {
		return fmt.Sprintf("%f", f*float64(n))
	}
}

func labeler(label string, value int, f func(int) string) string {
	return fmt.Sprintf("%s - %d(%s)", label, value, f(value))
}

func main() {
	s := labeler("hello1", 100, convertToBin)
	fmt.Println(s)

	s = labeler("hello2", 100, convert2(123.456))
	fmt.Println(s)
}
