package main

import "fmt"

type PredicateKey int
type PredicateValue interface{}

const (
	Name PredicateKey = iota
	Street
	City
)

type Predicate struct {
	Key   PredicateKey
	Value PredicateValue
}

type PredicateFunc func(*Predicate)

func (p Predicate) String() string {
	return fmt.Sprintf("%v: %v", p.Key, p.Value)
}

func WithName(name string) PredicateFunc {
	return func(p *Predicate) {
		p.Key = Name
		p.Value = name
	}
}

func WithStreet(street string) PredicateFunc {
	return func(p *Predicate) {
		p.Key = Street
		p.Value = street
	}
}

func WithCity(city string) PredicateFunc {
	return func(p *Predicate) {
		p.Key = City
		p.Value = city
	}
}

func FindPredicates(predicates ...PredicateFunc) *Predicate {
	s := &Predicate{}
	for _, p := range predicates {
		p(s)
	}
	return s
}

func main() {

	p := FindPredicates(
		WithName("John"),
		WithStreet("Street"),
		WithCity("City"),
	)
	fmt.Println(p)

}
