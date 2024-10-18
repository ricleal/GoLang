package main

import (
	"fmt"
	"sort"
	"sync"

	"exp/fun/q1/fibs"
)

type Number interface {
	int | float64
}

func swap[T Number](a []T) {
	for i := 0; i < len(a)-1; i = i + 2 {
		a[i], a[i+1] = a[i+1], a[i]
	}
}

func fact(v int) int {
	var r int = 1
	for i := v; i > 0; i-- {
		r = r * i
	}
	return r
}

func sortMap(v map[int]string) {
	keys := make([]int, len(v))
	i := 0
	for k := range v {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	fmt.Println("map sorted")
	for i = 0; i < len(keys); i++ {
		fmt.Println("->", keys[i], ":", v[keys[i]])
	}
}

//

type MyStruct struct {
	F int
}

//

type Set struct {
	val map[int]struct{}
	m   sync.Mutex
}

func NewSet() *Set {
	return &Set{
		val: make(map[int]struct{}),
	}
}

func (s *Set) Add(v int) {
	defer s.m.Unlock()
	s.m.Lock()
	s.val[v] = struct{}{}
}

func (s *Set) Dump() []int {
	defer s.m.Unlock()
	s.m.Lock()
	r := []int{}
	for k := range s.val {
		r = append(r, k)
	}
	return r
}

func main() {
	// swap
	s := []int{1, 2, 3, 4, 5}
	fmt.Println("origin:", s)
	swap(s)
	fmt.Println("swap:", s)
	// fact
	v := fact(5)
	// sort map key
	fmt.Println("v:", v)
	m := map[int]string{
		1: "Ricardo",
		2: "Miguel",
		4: "Leal",
		3: "Ferraz",
	}
	sortMap(m)
	// test if key exists in map
	if val, ok := m[1]; ok {
		fmt.Println("Exists", val)
	}
	// type discoverer
	var x interface{}
	x = MyStruct{F: 123}
	if xx, ok := x.(MyStruct); ok {
		fmt.Println(xx.F)
	}
	// Set
	set := NewSet()
	set.Add(1)
	set.Add(2)
	set.Add(2)
	fmt.Println("set:", set.Dump())
	// fib
	for i := 0; i < 11; i++ {
		fmt.Printf("fib (%d)= %d\n", i, fibs.Fib(i))
		fmt.Printf("fib2(%d)= %d\n", i, fibs.Fib2(i))
		fmt.Printf("fib3(%d)= %d\n", i, fibs.Fib3(i))
	}
}
