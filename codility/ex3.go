package codility

import "math"

// A positive integer N is given. The goal is to find the highest power of 2 that divides N. In other words,
// we have to find the maximum K for which N modulo 2^K is 0.

// For example, given integer N = 24 the answer is 3, because 2^3 = 8 is the highest power of 2 that divides N.

// Write a function:

// func Solution(N int) int

// that, given a positive integer N, returns the highest power of 2 that divides N.

// For example, given integer N = 24, the function should return 3, as explained above.

func Solution3(N int) int {

	maxK := 0
	k := 0
	for {
		powerOf2 := math.Pow(2, float64(k))

		if powerOf2 > float64(N) {
			break
		}

		if N%int(powerOf2) == 0 {
			maxK = k
		}

		k++
	}
	return maxK
}
