package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// HTTP Parser
// JSON body can be a map or an array

func getURL(url string) (interface{}, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error getting URL: %w", err)
	}
	defer resp.Body.Close()

	var result interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("error decoding: %w", err)
	}
	return result, nil
}

func main() {
	// Example with JSON array endpoint
	arrayURL := "https://jsonplaceholder.typicode.com/users"

	fmt.Println("Fetching array data...")
	data, err := getURL(arrayURL)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Type switch directly on the result
	switch v := data.(type) {
	case []interface{}:
		fmt.Printf("Received a slice with %d elements\n", len(v))
		fmt.Printf("First element: %v\n", v[0])
	case map[string]interface{}:
		panic("Unexpected content")
	default:
		fmt.Printf("Unexpected type: %T\n", v)
	}

	fmt.Println("--------------------------")

	// Example with JSON object endpoint
	objectURL := "https://jsonplaceholder.typicode.com/users/1"

	fmt.Println("Fetching object data...")
	data2, err := getURL(objectURL)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Type switch directly on the result
	switch v := data2.(type) {
	case []interface{}:
		panic("Unexpected content")
	case map[string]interface{}:
		fmt.Printf("Received a map with %d keys\n", len(v))
		fmt.Printf("Keys: %v\n", getKeys(v))
	default:
		fmt.Printf("Unexpected type: %T\n", v)
	}
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
