package main

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

// https://stackoverflow.com/questions/49077516/encoding-with-little-endianness-go-lang/49081648#49081648

// When we write numbers in English, we write them in big-endian base-10 representation.
// For example the number "4567" is understood to mean 4*10^3 + 5*10^2 + 6*10^1 + 7*10^0.
// This is base-10 because each written digit differs in significance by a factor of 10 from adjacent digits,
// and it is big-endian because the first written digit is associated with the biggest power of 10.

// The same number 4567 could be written in little-endian base-10 as "7654",
// which in little-endian representation would mean 7*10^0 + 6*10^1 + 5*10^2 + 4*10^3,
// numerically the same as in the previous paragraph.
// This is little-endian because the first written digit is associated with the littlest power of 10.

// The binary.LittleEndian.Uint32 function receives a slice of bytes and reads out of it a 32-bit unsigned integer represented in little-endian base-256.

// So if the base-256 digits in the input slice b are 2,3,5,7 as they are in your code,
// the little-endian base-256 interpretation of those bytes is 2*256^0 + 3*256^1 + 5*256^2 + 7*256^3.
// The same number written in big-endian base-10 (which is what fmt.Printf will show you) is "117768962".

func checkEndianess() binary.ByteOrder {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = uint16(0xABCD)

	switch buf {
	case [2]byte{0xCD, 0xAB}:
		return binary.LittleEndian
	case [2]byte{0xAB, 0xCD}:
		return binary.BigEndian
	default:
		panic("Could not determine native endianness.")
	}
}

func convertStringToFloat32(s string) float32 {
	// We assume that the number always has a decimal point

	// conert string to bytes
	bs := []byte(s)

	isNegative := false
	if bs[0] == '-' {
		isNegative = true
		bs = bs[1:]
	}

	var f float32
	var multiplier float32 = 10.0
	for i := 0; i < len(bs); i++ {
		digit := bs[i]
		if digit == '.' {
			continue
		}
		digit -= '0'

		f *= multiplier
		f += float32(digit)
	}

	f /= 10

	if isNegative {
		f *= -1
	}
	return f
}

func main() {
	// check the endianness of the system
	endianess := checkEndianess()
	fmt.Println("Endianness: ", endianess)

	numbers := []string{"-89.9", "-45.5", "-12.2", "-3.0", "0.0", "3.0", "12.2", "45.5", "89.9"}

	for _, number := range numbers {
		f := convertStringToFloat32(number)
		fmt.Println("Number: ", number, "=>", f)
	}
}
