package main

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
	ViolatorsEnter map[string]interface{}
	ViolatorsExit  map[string]interface{}
)

var LastOperation map[string]BadgingOperation

func main() {
	// Initialize maps
	ViolatorsEnter = make(map[string]interface{})
	ViolatorsExit = make(map[string]interface{})
	LastOperation = make(map[string]BadgingOperation)

	for _, entry := range badgeRecords {
		name, operation := entry[0], entry[1]

		lastOperation, found := LastOperation[name]
		if !found {
			// Expect a enter
			if BadgingOperation(operation) != Enter {
				ViolatorsEnter[name] = struct{}{}
				continue
			}
			LastOperation[name] = BadgingOperation(operation)
			continue
		}

		if lastOperation == BadgingOperation(operation) {
			// Violation
			if BadgingOperation(operation) == Enter {
				ViolatorsExit[name] = struct{}{}
			} else {
				ViolatorsEnter[name] = struct{}{}
			}
		}

		LastOperation[name] = BadgingOperation(operation)

	}

	for k := range ViolatorsEnter {
		println("Violator Enter:", k)
	}
	for k := range ViolatorsExit {
		println("Violator Exit:", k)
	}
}
