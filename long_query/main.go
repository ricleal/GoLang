package main

import (
	"fmt"
	"sync"
	"time"
)

func longQuery(query string) string {
	time.Sleep(2 * time.Second)
	return query
}

func longFunction(f func(string) string) {
	resultChan := make(chan string)
	var wg sync.WaitGroup

	// Run the long query in a goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		res := f("some query")
		fmt.Println("Query done.")
		resultChan <- res
	}()

	// Print "Query alive" messages until the query is done
	for {
		select {
		case <-resultChan:
			close(resultChan)
			wg.Wait()
			return
		default:
			fmt.Println("Query alive...")
			time.Sleep(300 * time.Millisecond)
		}
	}

}

func main() {
	fmt.Print("Running...")
	go longFunction(longQuery)
	time.Sleep(5 * time.Second)
	fmt.Print("Done!")
}
