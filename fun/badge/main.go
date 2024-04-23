package main

import "fmt"

// Find the people who failed to badge in and out correctly

var badgeRecords = [][]string{
	{"Martha", "exit"},
	{"Paul", "enter"},
	{"Martha", "enter"},
	{"Martha", "exit"},
	{"Ricardo", "enter"},
	{"Jennifer", "enter"},
	{"Paul", "enter"},
	{"Curtis", "enter"},
	{"Paul", "exit"},
	{"Martha", "enter"},
	{"Ricardo", "exit"},
	{"Ricardo", "enter"},
	{"Martha", "exit"},
	{"Jennifer", "exit"},
	{"Paul", "exit"},
	{"Ricardo", "exit"},
	{"Ricardo", "exit"},
	{"Ricardo", "enter"},
}

type BadgingOperation string

const (
	Enter BadgingOperation = "enter"
	Exit  BadgingOperation = "exit"
)

var (
	ViolatorsExit  = make(map[string]struct{})
	ViolatorsEnter = make(map[string]struct{})
)

func main() {
	lastOperations := make(map[string]BadgingOperation)

	for _, record := range badgeRecords {
		name := record[0]
		operation := BadgingOperation(record[1])

		// Check if this person has badged already
		if v, ok := lastOperations[name]; ok && v == operation {
			// same operation => violation
			if operation == Enter {
				ViolatorsExit[name] = struct{}{}
			} else {
				ViolatorsEnter[name] = struct{}{}
			}
		} else if !ok && operation == Exit {
			// First time badge
			ViolatorsEnter[name] = struct{}{}
		}
		lastOperations[name] = operation
	}

	// if the last operation is Enter, this person did not badge out
	for name, operation := range lastOperations {
		if operation == Enter {
			ViolatorsExit[name] = struct{}{}
		}
	}

	// Print results
	fmt.Printf("Violators who did not badge in:\n\t")
	for name := range ViolatorsEnter {
		fmt.Print(name)
		fmt.Print(" ")
	}
	fmt.Println()

	fmt.Printf("Violators who did not badge out:\n\t")
	for name := range ViolatorsExit {
		fmt.Print(name)
		fmt.Print(" ")
	}
}
