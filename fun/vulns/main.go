package main

import (
	"fmt"
	"os"
	"strings"
)

var breach1 = []string{
	"hello@example1.com",
	"hello1@example1.com",
	"hello2@example1.com",
	"hello3@example1.com",
	"world@example2.com",
}

var breach2 = []string{
	"hello@example1.com",
	"world@example2.com",
	"world1@example2.com",
	"world2@example2.com",
	"world3@example2.com",
}

var breaches = make(map[string]map[string][]string)

func loadData() {
	for idxBreach, breach := range [][]string{breach1, breach2} {
		for _, email := range breach {
			parts := strings.Split(email, "@")
			if len(parts) != 2 {
				continue
			}
			domain := parts[1]
			if _, ok := breaches[domain]; !ok {
				breaches[domain] = make(map[string][]string)
			}
			breaches[domain][email] = append(breaches[domain][email], fmt.Sprintf("breach%d", idxBreach+1))
		}
	}
}

func findBreach(domain string) map[string][]string {
	if emails, ok := breaches[domain]; ok {
		return emails
	}
	return nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: main <domain>")
		return
	}
	loadData()

	domain := os.Args[1]

	m := findBreach(domain)
	for k, v := range m {
		fmt.Printf("%s: %v\n", k, v)
	}
}
