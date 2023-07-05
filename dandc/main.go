package main

import (
	"context"
	"fmt"
)

type Number interface {
	int | float64
}

// Divide and Conquer

// serial
func search[N Number](ctx context.Context, data []N, key N) bool {
	if len(data) == 1 {
		return key == data[0]
	}
	mid := len(data) / 2
	return (search[N](ctx, data[:mid], key) ||
		search[N](ctx, data[mid:], key))
}

// parallel
func searchParallel[N Number](ctx context.Context, data []N, key N) bool {
	// TODO: implement a parallel search
	return false
}

func main() {
	ctx := context.Background()
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	found := search[int](ctx, data, 5)
	if found {
		fmt.Printf("found\n")
	} else {
		fmt.Printf("not found\n")
	}
	//
}
