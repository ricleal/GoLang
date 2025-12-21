package main

import (
	"fmt"
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

type Breaches map[string][]string // map indexed by emails and list of breaches

var DomainBreaches = map[string]Breaches{}

func main() {
	breaches := map[string][]string{
		"breach1": breach1,
		"breach2": breach2,
	}

	for breachName, emails := range breaches {
		for _, email := range emails {
			parts := strings.Split(email, "@")
			if len(parts) != 2 {
				panic("expectedd length is 2")
			}
			name := parts[0]
			domain := parts[1]
			if DomainBreaches[domain] == nil {
				DomainBreaches[domain] = make(map[string][]string)
			}
			DomainBreaches[domain][name] = append(DomainBreaches[domain][name], breachName)

		}
	}

	for domain, breaches := range DomainBreaches {
		fmt.Printf("** Breaches for domain: %s**\n", domain)
		for email, breachList := range breaches {
			fmt.Printf("%s: %v\n", email, breachList)
		}
	}
}
