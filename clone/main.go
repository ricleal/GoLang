package main

type Person interface {
	Clone() Person
	Greet(string) string
}

type person struct {
	name string
}

// Must implement Clone() and returning a pointer to the struct person,
// is not possible: func (p *person) Clone() *person {
func (p *person) Clone() Person {
	return &person{name: p.name}
}

func (p *person) Greet(name string) string {
	return "Hello " + name + ", my name is " + p.name
}

func main() {
	var p1 Person = &person{name: "John"}
	p2 := p1.Clone()
	println(p1.Greet("World 1"))
	println(p2.Greet("World 2"))
}
