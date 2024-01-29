package main

import (
	"log"
	"net/http"
	"sync"
)

func main() {

	client := http.Client{}
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			log.Printf("Sending request %d", i)
			defer wg.Done()
			resp, err := client.Get("http://localhost:8887")
			if err != nil {
				panic(err)
			}
			resp.Body.Close()
			// Convert body to string
			b := make([]byte, 100)
			resp.Body.Read(b)
			body := string(b)
			log.Printf("Response %d (%d): %s", i, resp.StatusCode, body)
		}(i)
	}

	log.Printf("Waiting for goroutines to finish")
	wg.Wait()
}
