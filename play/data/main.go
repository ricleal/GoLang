package main

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"sync"

	"github.com/google/uuid"
)

type Employee struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
}

type EmployeeSimple struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func produce(ctx context.Context, out chan<- Employee) {
	file, err := os.Open("employees.jsonl")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var emp Employee
		if err := json.Unmarshal([]byte(line), &emp); err != nil {
			panic(err)
		}
		select {
		case out <- emp:
		case <-ctx.Done():
			return
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

type EmployeeRecords struct {
	m       *sync.Mutex
	records map[uuid.UUID]EmployeeSimple
}

func NewEmployeeRecords() *EmployeeRecords {
	return &EmployeeRecords{
		m:       &sync.Mutex{},
		records: make(map[uuid.UUID]EmployeeSimple),
	}
}

func (e *EmployeeRecords) Add(emp Employee) {
	e.m.Lock()
	defer e.m.Unlock()
	e.records[emp.ID] = EmployeeSimple{
		FirstName: emp.FirstName,
		LastName:  emp.LastName,
	}
}

func consume(ctx context.Context, in <-chan Employee, employeeRecords *EmployeeRecords) {
	for {
		select {
		case emp, ok := <-in:
			if !ok {
				return // Channel closed
			}
			println(emp.FirstName, emp.LastName)
			employeeRecords.Add(emp)

		case <-ctx.Done():
			return
		}
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	employeeRecords := NewEmployeeRecords()

	employeeChan := make(chan Employee)

	var wg sync.WaitGroup
	wg.Go(func() {
		produce(ctx, employeeChan)
	})

	var wgConsumer sync.WaitGroup
	// launch 3 consumers
	for i := 0; i < 3; i++ {
		wgConsumer.Add(1)
		go func() {
			defer wgConsumer.Done()
			consume(ctx, employeeChan, employeeRecords)
		}()
	}

	wg.Wait()
	close(employeeChan)
	wgConsumer.Wait()

	// Print the employee records
	for id, emp := range employeeRecords.records {
		println(id.String(), emp.FirstName, emp.LastName)
	}
}
