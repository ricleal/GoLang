package main

import "fmt"

type MyStruct struct {
	Name  string
	Power int
}

func Super(s *MyStruct) {
	s.Power += 10000
}

func (s *MyStruct) Super() {
	s.Power += 10000
}

func main() {

	ricardo := MyStruct{"Ricardo", 9000}

	Super(&ricardo)
	fmt.Println(ricardo.Power)

	ricardo.Super()
	fmt.Println(ricardo.Power)

}
