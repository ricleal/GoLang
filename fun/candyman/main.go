package main

// The random Candyman

// The random Candyman puts candy sticks in his candy bag. They have various sizes and can be green or red.
// When someone asks for a candy stick, the Candyman randomly gets one from his bag.

// The Candyman would like to inform people about the average size of the following candy stick that will be chosen and the chance to have a red one.
// But he wants to keep his bag contents private and may be too lazy to inspect the bag by himself. Fortunately, the Candyman knows basic mathematics and can remember a few numbers.

// How can the Candyman inform people about his candies?

// You have to implement a CandyMan class or structure containing a list of candy sticks. The class must have the methods:

// add_candy
// get_a_random_candy
// get_average_size
// get_red_candy_chance.

import (
	"fmt"
	"math/rand"
)

type CandyColor string

const (
	CandlyColorRed  CandyColor = "red"
	CandyColorGreen CandyColor = "green"
)

type Candy struct {
	color CandyColor
	size  int
}

type CandyMan struct {
	bag []Candy
}

func (c *CandyMan) AddCandy(candy Candy) {
	c.bag = append(c.bag, candy)
}

func (c *CandyMan) RandomCandy() *Candy {
	r := rand.Intn(len(c.bag))
	return &c.bag[r]
}

func (c *CandyMan) AverageSize() float32 {
	sum := 0
	for _, c := range c.bag {
		sum += c.size
	}
	return float32(sum) / float32(len(c.bag))
}

func (c *CandyMan) RedCandyChance() float32 {
	return 0.5
}

/////////////////////

type Observer interface {
	OnNotify(c CandyMan)
}

type Notifier interface {
	Register(o Observer)
	Unregister(o Observer)
	Notitfy(c CandyMan)
}

type observer struct {
}

func (o *observer) OnNotify(c CandyMan) {
	fmt.Println("Candy", c.AverageSize(), c.RedCandyChance())
}

type notifier struct {
	registry map[Observer]struct{}
}

func (r *notifier) Register(o Observer) {
	r.registry[o] = struct{}{}
}

func (r *notifier) Unregister(o Observer) {
	delete(r.registry, o)
}

func (r *notifier) Notify(c CandyMan) {
	for k := range r.registry {
		k.OnNotify(c)
	}
}

func main() {
	o1 := observer{}
	o2 := observer{}
	n := notifier{
		registry: map[Observer]struct{}{},
	}
	n.Register(&o1)
	n.Register(&o2)

	c := CandyMan{}
	c.AddCandy(Candy{color: CandlyColorRed, size: 1})
	c.AddCandy(Candy{color: CandlyColorRed, size: 2})
	c.AddCandy(Candy{color: CandyColorGreen, size: 3})
	n.Notify(c)

}
