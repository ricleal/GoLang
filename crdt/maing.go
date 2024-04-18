package main

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"sync"

	"github.com/google/uuid"
)

type Message struct {
	ID         uuid.UUID `json:"id"`
	Increments int       `json:"increments"`
	Decrements int       `json:"decrements"`
}

type CRDTCounter struct {
	ID              uuid.UUID
	mu              sync.Mutex
	increments      int
	decrements      int
	otherIncrements map[uuid.UUID]int
	otherDecrements map[uuid.UUID]int
}

func NewCRDTCounter() *CRDTCounter {
	return &CRDTCounter{
		ID:              uuid.New(),
		otherIncrements: make(map[uuid.UUID]int),
		otherDecrements: make(map[uuid.UUID]int),
	}
}

func (c *CRDTCounter) Inc() {
	c.increments++
}

func (c *CRDTCounter) Dec() {
	c.decrements++
}

func (c *CRDTCounter) Value() int {
	return c.increments - c.decrements
}

func (c *CRDTCounter) EmitStatus() *Message {
	return &Message{
		ID:         c.ID,
		Increments: c.increments,
		Decrements: c.decrements,
	}
}

func (c *CRDTCounter) ReceiveStatus(m *Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.otherIncrements[m.ID] <= m.Increments {
		c.otherIncrements[m.ID] = m.Increments
	}
	if c.otherDecrements[m.ID] <= m.Decrements {
		c.otherDecrements[m.ID] = m.Decrements
	}
}

func (c *CRDTCounter) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := "----------------------------------------------------\n"
	s += c.ID.String()
	s += "\n"
	s += fmt.Sprintf("inc: %d, dec: %d, value: %d\n", c.increments, c.decrements, c.Value())
	s += "Other Increments:\n"
	for k, v := range c.otherIncrements {
		s += "\t" + k.String() + ": " + strconv.Itoa(v) + "\n"
	}
	s += "Other Decrements:\n"
	for k, v := range c.otherDecrements {
		s += "\t" + k.String() + ": " + strconv.Itoa(v) + "\n"
	}
	s += "Other Values:\n"
	for k := range c.otherIncrements {
		v := c.otherIncrements[k] - c.otherDecrements[k]
		s += "\t" + k.String() + ": " + strconv.Itoa(v) + "\n"
	}
	s += "----------------------------------------------------\n"
	return s
}

func main() {
	// set a random seed for the random number generator
	r := rand.New(rand.NewPCG(123, 456))

	// let's simulate CRDTCounter working
	nCRDTs := 10
	register := make([]*CRDTCounter, 0, nCRDTs)
	for i := 0; i < nCRDTs; i++ {
		register = append(register, NewCRDTCounter())
	}

	uuids := make([]uuid.UUID, 0, nCRDTs)
	for i := 0; i < nCRDTs; i++ {
		uuids = append(uuids, register[i].ID)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// either increment or decrement
			randomOperation := r.IntN(2)
			crdtIndex := r.IntN(len(register))
			if randomOperation == 0 {
				register[crdtIndex].Inc()
			} else {
				register[crdtIndex].Dec()
			}
			msg := register[crdtIndex].EmitStatus()
			for j := 0; j < len(register); j++ {
				if j != crdtIndex {
					register[j].ReceiveStatus(msg)
				}
			}
		}()
	}
	wg.Wait()
	// let's print the status
	for i := 0; i < nCRDTs; i++ {
		fmt.Println(register[i].String())
	}
	// make sure main CRDTCounter has the same value as the others
	for i := 0; i < nCRDTs; i++ {
		for j := 0; j < nCRDTs; j++ {
			if i != j {
				for _, id := range uuids {
					if register[i].ID == id {
						v1 := register[i].Value()
						v2 := register[j].otherIncrements[id] - register[j].otherDecrements[id]
						if v1 != v2 {
							fmt.Println("❌ ERROR: Incorrect value for", i, j, id, v1, v2)
						} else {
							fmt.Println("✅", i, j, id, v1, v2)
						}
					}
				}
			}
		}
	}
	fmt.Println("✅")
	fmt.Println("Done!")
}
