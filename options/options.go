package main

import "fmt"

type S struct {
	opt1 string
	opt2 int
}

func (s *S) String() string {
	return fmt.Sprintf("S{opt1: %q, opt2: %d}", s.opt1, s.opt2)
}

type Option func(*S) error // HL

func Opt1(v string) Option {
	return func(s *S) error {
		s.opt1 = v
		return nil
	}
}

func Opt2(v int) Option {
	return func(s *S) error {
		s.opt2 = v
		return nil
	}
}

func New(opts ...Option) (*S, error) {
	s := &S{}
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func main() {
	s, err := New(Opt1("hello"), Opt2(42))
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", s)
	// Output: &main.S{opt1: "hello", opt2: 42}
	fmt.Println(s)
	// Output: S{opt1: "hello", opt2: 42}
	_ = s
}
