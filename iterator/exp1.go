package main

import (
	"fmt"
	"sync"
)

// see: https://antonz.org/go-1-23/

type data struct {
	key   string
	value string
}

func expMap() {
	var m sync.Map

	m.Store("key1", data{"key1", "value1"})
	m.Store("key2", data{"key2", "value2"})
	m.Store("key3", data{"key3", "value3"})
	m.Store("key4", data{"key4-inv", "value4"})
	m.Store("key5", data{"key5", "value5"})

	m.Range(func(key, value any) bool {
		// cast key to string
		if keyStr, ok := key.(string); ok {
			// cast value to data
			if valueData, ok := value.(data); ok {
				if valueData.key == keyStr {
					fmt.Printf("✅ key: %s, value: %s\n", keyStr, valueData.value)
				} else {
					fmt.Println("❗key and value not match")
				}
			}
		}
		return true
	})
}

func main() {
	expMap()
}
