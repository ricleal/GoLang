package main

import (
	"fmt"
	"strings"
)

type Employee struct {
	Id        int
	ManagerId int
	Name      string
}

var EMPLOYEES = []Employee{
	{Id: 2, ManagerId: 8, Name: "Bob"},
	{Id: 3, ManagerId: 2, Name: "Emp3"},
	{Id: 4, ManagerId: 3, Name: "Emp4"},
	{Id: 8, ManagerId: 8, Name: "Alice"},
	{Id: 5, ManagerId: 4, Name: "Emp5"},
	{Id: 6, ManagerId: 3, Name: "Emp6"},
	{Id: 7, ManagerId: 8, Name: "Emp7"},
}

func printHierarchy() {
	// Build a map of managerId to list of employees
	managerMap := make(map[int][]Employee)
	employeeMap := make(map[int]Employee)

	for _, emp := range EMPLOYEES {
		managerMap[emp.ManagerId] = append(managerMap[emp.ManagerId], emp)
		employeeMap[emp.Id] = emp
	}

	// Find the top-level manager (self-managing employee)
	var topManager Employee
	for _, emp := range EMPLOYEES {
		if emp.Id == emp.ManagerId {
			topManager = emp
			break
		}
	}

	// Print the hierarchy recursively
	printEmployee(topManager, 0, managerMap)
}

func printEmployee(emp Employee, level int, managerMap map[int][]Employee) {
	indent := strings.Repeat("  ", level)
	if level == 0 {
		fmt.Printf("%s%s (%d)\n", indent, emp.Name, emp.Id)
	} else {
		fmt.Printf("%s%s\n", indent, emp.Name)
	}

	// Print all direct reports
	reports := managerMap[emp.Id]
	for _, report := range reports {
		if report.Id != emp.Id { // Skip self-reference
			printEmployee(report, level+1, managerMap)
		}
	}
}

func main() {
	printHierarchy()
}
