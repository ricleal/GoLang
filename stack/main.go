package main

type Stack[T any] []T

func (s *Stack[T]) Push(v T) {
	*s = append(*s, v)
}

func (s *Stack[T]) Pop() T {
	l := len(*s)
	v := (*s)[l-1]
	*s = (*s)[:l-1]
	return v
}

func (s Stack[T]) Peek() T {
	return s[len(s)-1]
}

func (s *Stack[T]) Len() int {
	return len(*s)
}

func main() {
	var s Stack[int]
	s.Push(1)
	s.Push(2)
	s.Push(3)
	println(s.Pop())
	println(s.Pop())
	println(s.Pop())
}
