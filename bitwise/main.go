package main

import (
	"fmt"
	"math"
)

func main() {
	fmt.Printf("2<<1 => bin: %08b hex: %x dec: %d\n", 2<<1, 2<<1, 2<<1)
	fmt.Printf("2<<2 => bin: %08b hex: %x dec: %d\n", 2<<2, 2<<2, 2<<2)
	fmt.Printf("2<<3 => bin: %08b hex: %x dec: %d\n", 2<<3, 2<<3, 2<<3)
	fmt.Printf("2<<4 => bin: %08b hex: %x dec: %d\n", 2<<4, 2<<4, 2<<4)
	fmt.Printf("2<<5 => bin: %08b hex: %x dec: %d\n", 2<<5, 2<<5, 2<<5)

	fmt.Printf("2**1 => bin: %08b hex: %x dec: %d\n",
		int(math.Pow(float64(2), float64(1))),
		int(math.Pow(float64(2), float64(1))),
		int(math.Pow(float64(2), float64(1))))
	fmt.Printf("2**2 => bin: %08b hex: %x dec: %d\n",
		int(math.Pow(float64(2), float64(2))),
		int(math.Pow(float64(2), float64(2))),
		int(math.Pow(float64(2), float64(2))))
	fmt.Printf("2**3 => bin: %08b hex: %x dec: %d\n",
		int(math.Pow(float64(2), float64(3))),
		int(math.Pow(float64(2), float64(3))),
		int(math.Pow(float64(2), float64(3))))
	fmt.Printf("2**4 => bin: %08b hex: %x dec: %d\n",
		int(math.Pow(float64(2), float64(4))),
		int(math.Pow(float64(2), float64(4))),
		int(math.Pow(float64(2), float64(4))))
	fmt.Printf("2**5 => bin: %08b hex: %x dec: %d\n",
		int(math.Pow(float64(2), float64(5))),
		int(math.Pow(float64(2), float64(5))),
		int(math.Pow(float64(2), float64(5))))
}
