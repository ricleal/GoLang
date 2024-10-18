package fibs

// fibonacci
// n  =	0	1	2	3	4	5	6	7	  8	  9	  10	11	12	13	14	...
// xn =	0	1	1	2	3	5	8	13	21	34	55	89	144	233	377	...
// xn = xn−1 + xn−2
// fibonacci recursive.
func Fib(n int) int {
	if n == 0 {
		return 0
	} else if n == 1 {
		return 1
	}
	return Fib(n-1) + Fib(n-2)
}

// fibonacci iterative.
func Fib2(n int) int {
	if n == 0 {
		return 0
	} else if n == 1 {
		return 1
	}

	results := make([]int, n+1)
	results[0] = 0
	results[1] = 1

	for i := 2; i <= n; i++ {
		results[i] = results[i-1] + results[i-2]
	}
	return results[n]
}

// fibonacci iterative.
func Fib3(n int) int {
	if n == 0 {
		return 0
	} else if n == 1 {
		return 1
	}

	a := 0
	b := 1

	var c int
	for i := 2; i <= n; i++ {
		c = b + a
		a = b
		b = c
	}
	return c
}

var cache = map[int]int{}

// fibonacci recursive with cache.
func Fib4(n int) int {
	if n == 0 {
		return 0
	} else if n == 1 {
		return 1
	}
	if v, ok := cache[n]; ok {
		return v
	}
	cache[n] = Fib4(n-1) + Fib4(n-2)
	return cache[n]
}
