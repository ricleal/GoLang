package main

import (
	"fmt"
	"log"
)

func inner(innerParam string) error {
	if innerParam == "" {
		return fmt.Errorf("innerParam cannot be empty")
	}
	log.Println("innerFunc called with:", innerParam)
	return nil
}

type innerFunc func(innerParam string) error

func outer(outerParam int, innerFunc innerFunc) error {
	if outerParam < 0 {
		return fmt.Errorf("outerParam cannot be negative")
	}
	log.Println("outerFunc called with:", outerParam)
	innerParam := fmt.Sprintf("innerParam=%d", outerParam)
	return innerFunc(innerParam)
}

func main() {
	// outerFunc called with: 10
	if err := outer(10, inner); err != nil {
		log.Println("Error calling outer:", err)
	}

	// outerFunc called with: 11 inline innerFunc
	if err := outer(11, func(s string) error {
		log.Println("inline innerFunc called with:", s)
		return nil
	}); err != nil {
		log.Println("Error calling outer:", err)
	}
}
