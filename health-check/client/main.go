package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	serviceName := "my-service"

	// First register service
	resp, err := http.Post("http://localhost:8080/register/"+serviceName, "", http.NoBody)
	if err != nil {
		fmt.Println("Service registration failed!")
		os.Exit(1)
	}
	var m map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&m)
	resp.Body.Close()
	fmt.Println("Response:", m)
	fmt.Println("Service registered successfully!")

	go func() {
		for {
			resp, err := http.Post("http://localhost:8080/heartbeat/"+serviceName, "", http.NoBody)
			if err != nil {
				fmt.Println("Service is not responding...")
			} else {
				fmt.Println("Service is alive! Status code:", resp.StatusCode)
				// Decode response into map
				var m map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&m)
				resp.Body.Close()
				// print response
				fmt.Println("Response:", m)
			}
			time.Sleep(1 * time.Second) // Send heartbeat every second
			// check ctx.Done() here
			select {
			case <-ctx.Done():
				fmt.Println("Shutting down heartbeat goroutine...")
				return
			default:
				// Do nothing
			}
		}
	}()

	// Simulate service doing its work
	for {
		fmt.Println("Service is working...")
		time.Sleep(2 * time.Second)
		// check ctx.Done() here
		select {
		case <-ctx.Done():
			fmt.Println("Shutting down heartbeat goroutine...")
			return
		default:
			// Do nothing
		}
	}
}
