package main

import "fmt"

func main() {
	scores := [4]int{9001, 9333, 212, 33}

	for index, value := range scores {
		fmt.Print(index, value, "; ")
	}
	fmt.Println()

	scores2 := make([]int, 10, 10) //len, capacity
	scores2[7] = 9033
	fmt.Println(scores2)

	names := []string{"leto", "jessica", "paul"}
	fmt.Println(names)
	checks := make([]bool, 10)
	fmt.Println(checks)
	var names2 []string
	fmt.Println(names2)
	scores3 := make([]int, 0, 20)
	fmt.Println(scores3)

}
