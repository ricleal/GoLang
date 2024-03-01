package main

import (
	"fmt"
)

type A1 struct {
	Name string
}

func (a *A1) do() {
	fmt.Println("Name:", a.Name)
}

type B1 struct {
	A1
	Name string
}

// Implement a Stack (LIFO)
type Stack struct {
	contents []int
}

func (s *Stack) Push(v int) {
	s.contents = append(s.contents, v)
}

func (s *Stack) Pop() int {
	if len(s.contents) == 0 {
		panic("Empty stack")
	}
	v := s.contents[len(s.contents)-1]
	s.contents = s.contents[:len(s.contents)-1]
	return v
}

// printing numbers from 1 to 100, but with some exceptions.
// "Fizz" for multiples of 3,
// "Buzz" for multiples of 5,
// "FizzBuzz" for multiples of both 3 and 5.
func fizzbuzz() {
	for n := 1; n <= 100; n++ {
		if n%3 == 0 && n%5 == 0 {
			fmt.Println(n, "FizzBuzz")
		} else if n%3 == 0 {
			fmt.Println(n, "Fizz")
		} else if n%5 == 0 {
			fmt.Println(n, "Buzz")
		} else {
			fmt.Println(n)
		}
	}
}

func fizzbuzz2() {
	for n := 1; n <= 100; n++ {
		out := fmt.Sprintf("%d ", n)
		if n%3 == 0 {
			out = out + "Fizz"
		}
		if n%5 == 0 {
			out = out + "Buzz"
		}
		fmt.Println(out)
	}
}

// Queue
type queue struct {
	contents []interface{}
}

func (q *queue) enqueue(v interface{}) {
	q.contents = append(q.contents, v)
}

func (q *queue) dequeue() interface{} {
	if len(q.contents) == 0 {
		return nil
	}
	v := q.contents[0]
	// Does it creates a new slice
	q.contents = q.contents[1:]
	return v
}

func main() {
	b := B1{
		A1: A1{
			Name: "bFromA1",
		},
		Name: "b",
	}
	b.do()
	// stack
	s := Stack{}
	s.Push(1)
	s.Push(2)
	fmt.Println("pop", s.Pop())
	fmt.Println("pop", s.Pop())
	// fmt.Println("pop", s.Pop()) // panic
	// fizzbuzz2()
	q := queue{}
	q.enqueue(1)
	q.enqueue(2)
	_ = q.dequeue()
	q.enqueue(3)
}
