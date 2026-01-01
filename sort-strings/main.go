package main

import (
	"fmt"
)

func lengthOfLongestSubstring(s string) int {
	var max int
	var start int
	var end int
	m := make(map[byte]int)
	for i := 0; i < len(s); i++ {
		if v, ok := m[s[i]]; ok && v >= start {
			start = v + 1
		}
		m[s[i]] = i
		end = i
		if end-start+1 > max {
			max = end - start + 1
		}
	}
	return max
}

func findMedianSortedArrays(nums1 []int, nums2 []int) float64 {
	var i, j int
	var result []int
	for i < len(nums1) && j < len(nums2) {
		if nums1[i] < nums2[j] {
			result = append(result, nums1[i])
			i++
		} else {
			result = append(result, nums2[j])
			j++
		}
	}
	if i < len(nums1) {
		result = append(result, nums1[i:]...)
	}
	if j < len(nums2) {
		result = append(result, nums2[j:]...)
	}
	if len(result)%2 == 0 {
		return float64(result[len(result)/2-1]+result[len(result)/2]) / 2
	} else {
		return float64(result[len(result)/2])
	}
}

// bubbleSort sorts a slice of integers using bubble sort algorithm
// Time Complexity: O(n²) - not efficient for large datasets
// Space Complexity: O(1) - sorts in place
//
// How it works:
//
//	Pass 1: [64, 34, 25, 12]  →  compare adjacent pairs, swap if needed
//	        [34, 64, 25, 12]  →  64 > 34, swap
//	        [34, 25, 64, 12]  →  64 > 25, swap
//	        [34, 25, 12, 64]  →  64 > 12, swap (largest bubbles to end)
//
//	Pass 2: [34, 25, 12, 64]  →  ignore last element (already sorted)
//	        [25, 34, 12, 64]  →  34 > 25, swap
//	        [25, 12, 34, 64]  →  34 > 12, swap
//
//	Pass 3: [25, 12, 34, 64]  →  ignore last 2 elements
//	        [12, 25, 34, 64]  →  25 > 12, swap
//
//	Result: [12, 25, 34, 64]  →  fully sorted!
func bubbleSort(arr []int) []int {
	n := len(arr)
	// Make a copy to avoid modifying the original
	sorted := make([]int, n)
	copy(sorted, arr)

	for i := 0; i < n-1; i++ {
		// Flag to optimize by stopping if no swaps occur
		swapped := false

		// Last i elements are already in place
		for j := 0; j < n-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				// Swap elements
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
				swapped = true
			}
		}

		// If no swaps occurred, array is already sorted
		if !swapped {
			break
		}
	}

	return sorted
}

// mergeSort sorts a slice of integers using merge sort algorithm
// Time Complexity: O(n log n) - efficient for large datasets
// Space Complexity: O(n) - requires additional space for merging
//
// How it works (Divide and Conquer):
//
//	Original: [64, 34, 25, 12]
//
//	DIVIDE:
//	                 [64, 34, 25, 12]
//	                /                \
//	         [64, 34]                [25, 12]
//	          /     \                 /     \
//	       [64]    [34]            [25]    [12]
//
//	CONQUER (Merge):
//	          |       |               |       |
//	         [34, 64]                [12, 25]    ← merge sorted halves
//	                \                /
//	               [12, 25, 34, 64]              ← final merge
//
//	Example merge of [34, 64] and [12, 25]:
//	  Compare: 34 vs 12 → take 12  → [12]
//	  Compare: 34 vs 25 → take 25  → [12, 25]
//	  Take remaining: 34, 64       → [12, 25, 34, 64]
func mergeSort(arr []int) []int {
	// Base case: array with 0 or 1 element is already sorted
	if len(arr) <= 1 {
		return arr
	}

	// Divide: split array into two halves
	mid := len(arr) / 2
	left := mergeSort(arr[:mid])
	right := mergeSort(arr[mid:])

	// Conquer: merge the sorted halves
	return merge(left, right)
}

// merge combines two sorted slices into one sorted slice
func merge(left, right []int) []int {
	result := make([]int, 0, len(left)+len(right))
	i, j := 0, 0

	// Compare elements from left and right, append smaller one
	for i < len(left) && j < len(right) {
		if left[i] <= right[j] {
			result = append(result, left[i])
			i++
		} else {
			result = append(result, right[j])
			j++
		}
	}

	// Append remaining elements from left or right
	result = append(result, left[i:]...)
	result = append(result, right[j:]...)

	return result
}

func main() {
	fmt.Println("=== String Algorithms ===")
	fmt.Printf("Longest substring without repeating: %d\n\n", lengthOfLongestSubstring("aab"))

	fmt.Println("=== Sorting Algorithms ===")

	// Test data
	unsorted := []int{64, 34, 25, 12, 22, 11, 90, 88, 45, 50, 23, 36}
	fmt.Printf("Original array: %v\n\n", unsorted)

	// Bubble Sort - visual explanation
	fmt.Println("┌─────────────────────────────────────────────────────────┐")
	fmt.Println("│ BUBBLE SORT - Repeatedly swap adjacent elements         │")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Println("│  Pass 1: [5, 3, 8, 2] → [3, 5, 8, 2] → [3, 5, 2, 8]     │")
	fmt.Println("│          ↑↓           ↑↓           ↑↓                   │")
	fmt.Println("│         swap          ok          swap (8 bubbles up)   │")
	fmt.Println("│                                                         │")
	fmt.Println("│  Pass 2: [3, 5, 2, 8] → [3, 2, 5, 8]                    │")
	fmt.Println("│             ↑↓         (ignore 8, already sorted)       │")
	fmt.Println("│                                                         │")
	fmt.Println("│  Pass 3: [3, 2, 5, 8] → [2, 3, 5, 8] ✓                  │")
	fmt.Println("│          ↑↓                                             │")
	fmt.Println("└─────────────────────────────────────────────────────────┘")
	bubbleSorted := bubbleSort(unsorted)
	fmt.Printf("\nBubble Sort Result: %v\n", bubbleSorted)
	fmt.Println("  Time Complexity: O(n²)")
	fmt.Println("  Space Complexity: O(1)")
	fmt.Println("  Best for: Small datasets or nearly sorted data")
	fmt.Println()

	// Merge Sort - visual explanation
	fmt.Println("┌─────────────────────────────────────────────────────────┐")
	fmt.Println("│ MERGE SORT - Divide and Conquer                         │")
	fmt.Println("├─────────────────────────────────────────────────────────┤")
	fmt.Println("│  DIVIDE:          [5, 3, 8, 2]                          │")
	fmt.Println("│                  /            \\                        │")
	fmt.Println("│              [5, 3]          [8, 2]                     │")
	fmt.Println("│              /    \\          /    \\                   │")
	fmt.Println("│            [5]    [3]      [8]    [2]                   │")
	fmt.Println("│                                                         │")
	fmt.Println("│  MERGE:       |      |      |      |                    │")
	fmt.Println("│             [3, 5]        [2, 8]    ← merge pairs       │")
	fmt.Println("│                \\            /                          │")
	fmt.Println("│               [2, 3, 5, 8] ✓        ← final merge       │")
	fmt.Println("└─────────────────────────────────────────────────────────┘")
	mergeSorted := mergeSort(unsorted)
	fmt.Printf("\nMerge Sort Result:  %v\n", mergeSorted)
	fmt.Println("  Time Complexity: O(n log n)")
	fmt.Println("  Space Complexity: O(n)")
	fmt.Println("  Best for: Large datasets, guaranteed performance")
}
