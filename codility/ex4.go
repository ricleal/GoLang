package codility

// An array A consisting of N integers is given. An inversion is a pair of indexes (P, Q) such that P < Q and A[Q] < A[P].

// Write a function:

// func Solution(A []int) int

// that computes the number of inversions in A, or returns −1 if it exceeds 1,000,000,000.

// For example, in the following array:

// A[0] = -1
// A[1] = 6
// A[2] = 3
// A[3] =  4
// A[4] = 7
// A[5] = 4

// there are four inversions:

// (1,2)  (1,3)  (1,5)  (4,5)
// so the function should return 4.

func Solution4(A []int) int {
	_, inversions := mergeSort(A)
	if inversions > 1000000000 {
		return -1
	}
	return inversions
}

func mergeSort(arr []int) ([]int, int) {
	if len(arr) < 2 {
		return arr, 0
	}
	mid := len(arr) / 2
	a, x := mergeSort(arr[:mid])
	b, y := mergeSort(arr[mid:])
	merged, inversions := merge(a, b)
	return merged, x + y + inversions
}

func merge(a, b []int) ([]int, int) {
	final := []int{}
	count := 0
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] <= b[j] {
			final = append(final, a[i])
			i++
		} else {
			final = append(final, b[j])
			j++
			count += len(a) - i
		}
	}
	for ; i < len(a); i++ {
		final = append(final, a[i])
	}
	for ; j < len(b); j++ {
		final = append(final, b[j])
	}
	return final, count
}
