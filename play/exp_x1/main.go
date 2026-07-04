package main

import "fmt"

type Set[T comparable] struct {
	value map[T]struct{}
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		value: make(map[T]struct{}),
	}
}

func (s *Set[T]) Add(value T) {
	s.value[value] = struct{}{}
}

func (s *Set[T]) Remove(value T) {
	delete(s.value, value)
}

func (s *Set[T]) Contains(value T) bool {
	_, exists := s.value[value]
	return exists
}

func (s *Set[T]) Size() int {
	return len(s.value)
}

func (s *Set[T]) Clear() {
	s.value = make(map[T]struct{})
}

///

type Stack[T comparable] struct {
	value []T
}

func NewStack[T comparable]() *Stack[T] {
	return &Stack[T]{
		value: []T{},
	}
}

func (s *Stack[T]) Push(e T) {
	s.value = append(s.value, e)
}

func (s *Stack[T]) Pop() (T, error) {
	if len(s.value) == 0 {
		return *new(T), fmt.Errorf("stack is empty")
	}
	lastIndex := len(s.value) - 1
	elem := s.value[lastIndex]
	s.value = s.value[:lastIndex]
	return elem, nil
}

func main() {
	set := NewSet[int]()
	set.Add(1)
	set.Add(2)
	set.Add(3)
	println(set.Contains(2)) // true
	println(set.Size())      // 3
	set.Remove(2)
	println(set.Contains(2)) // false
	set.Clear()
	println(set.Size()) // 0
}
