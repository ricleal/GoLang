package main

// Find the people who failed to badge in and out correctly

import "fmt"

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
	{"Ricardo", "enter"},
}

func failures(lst [][]string) (violEnter, violExit []string) {
	registry := make(map[string][]string)
	for _, l := range lst {
		registry[l[0]] = append(registry[l[0]], l[1])
	}
	fmt.Println(registry)

	for name, reg := range registry {
		for i, el := range reg {
			switch {
			case i == 0 && el == "exit":
				violEnter = append(violEnter, name)
			case i > 0 && reg[i-1] == "exit" && reg[i] == "exit":
				violEnter = append(violEnter, name)
			case i > 0 && reg[i-1] == "enter" && reg[i] == "enter":
				violExit = append(violExit, name)
			case len(reg)-1 == i && el == "enter":
				violExit = append(violExit, name)
			}
		}
	}
	return
}

func failures2(lst [][]string) (violEnter, violExit []string) {
	registry := make(map[string]string)
	for _, l := range lst {
		if _, ok := registry[l[0]]; !ok {
			if l[1] == "exit" {
				violEnter = append(violEnter, l[0])
				continue
			}
			registry[l[0]] = l[1]
			continue
		}
		if registry[l[0]] == l[1] {
			if l[1] == "enter" && registry[l[0]] == "enter" {
				violExit = append(violExit, l[0])
			} else {
				violEnter = append(violEnter, l[0])
			}
		}
		registry[l[0]] = l[1]
	}
	// Check for people who failed to badge out
	for name, reg := range registry {
		if reg == "enter" {
			violExit = append(violExit, name)
		}
	}
	return
}

func main() {
	violEnter, violExit := failures(badgeRecords)
	fmt.Println("Enter Violators:", violEnter)
	fmt.Println("Exit Violators: ", violExit)

	violEnter, violExit = failures2(badgeRecords)
	fmt.Println("Enter Violators:", violEnter)
	fmt.Println("Exit Violators: ", violExit)
}
