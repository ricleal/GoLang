package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

var (
	a = [][]int{{1, 2, 3}, {4, 5, 6}}
	b = [][]int{{7, 8}, {9, 10}, {11, 12}}
	c = [2][2]int{}
)

func multiplySerial() {
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			c[i][j] = 0
			for k := 0; k < 3; k++ {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}
}

func multiplyParallelErrGroup(_ context.Context) error {
	g := new(errgroup.Group)
	g.SetLimit(2) // Limit the number of goroutines to 2
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			g.Go(
				func(i, j int) func() error {
					return func() error {
						c[i][j] = 0
						for k := 0; k < 3; k++ {
							c[i][j] += a[i][k] * b[k][j]
						}
						return nil
					}
				}(i, j),
			)
		}
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func multiplyParallelSemaphores(ctx context.Context) {
	sem := semaphore.NewWeighted(2) // Limit the number of goroutines to 2
	wg := sync.WaitGroup{}

	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			wg.Add(1)
			go func(i, j int) {
				defer wg.Done()
				if err := sem.Acquire(ctx, 1); err != nil {
					// for simplicity, in case of error, we set the result to the minimum integer value
					for k := 0; k < 3; k++ {
						c[i][j] = math.MinInt
					}
					return
				}
				defer sem.Release(1)
				c[i][j] = 0
				for k := 0; k < 3; k++ {
					c[i][j] += a[i][k] * b[k][j]
				}
			}(i, j)
		}
	}
	wg.Wait()
}

func validateMatrix(a, b [][]int) error {
	if len(a) == 0 || len(b) == 0 {
		return errors.New("Matrix A and Matrix B must not be empty")
	}
	if len(a[0]) != len(b) {
		return errors.New("Number of columns in Matrix A must be equal to the number of rows in Matrix B")
	}
	return nil
}

func main() {
	if err := validateMatrix(a, b); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Matrix A")
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			fmt.Print(a[i][j], " ")
		}
		fmt.Println()
	}

	fmt.Println("Matrix B")
	for i := 0; i < 3; i++ {
		for j := 0; j < 2; j++ {
			fmt.Print(b[i][j], " ")
		}
		fmt.Println()
	}

	multiplySerial()

	fmt.Println("Serial: Matrix C")
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			fmt.Printf("%10d", c[i][j])
		}
		fmt.Println()
	}

	ctx := context.Background()
	multiplyParallelErrGroup(ctx)

	fmt.Println("Parallel ErrGroup: Matrix C")
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			fmt.Printf("%10d", c[i][j])
		}
		fmt.Println()
	}

	multiplyParallelSemaphores(ctx)

	fmt.Println("Parallel Semaphores: Matrix C")
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			fmt.Printf("%10d", c[i][j])
		}
		fmt.Println()
	}

	fmt.Println("Done")
}
