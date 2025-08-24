package main

import (
	"fmt"
	"net/http"
	"time"
)

const MaxTasks = 100

var sem = make(chan struct{}, MaxTasks)

func handler(w http.ResponseWriter, r *http.Request) {
	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
		// Simulate a task that takes time
		time.Sleep(time.Second)
		fmt.Fprintln(w, "Processed request")
	default:
		http.Error(w, "Server is under heavy load, please try again later", http.StatusServiceUnavailable)
	}
}

func main() {
	http.HandleFunc("/", handler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Error starting server:", err)
	} else {
		fmt.Println("Server started on :8080")
	}
}
