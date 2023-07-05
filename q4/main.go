package main

import "fmt"

func isPalindrome(s string) bool {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		if s[i] != s[j] {
			return false
		}
	}
	return true
}

func Max(arr []int) int {
	v := arr[0]
	for _, i := range arr {
		if i > v {
			v = i
		}
	}
	return v
}

// TODO: Implement the following functions:
// String Compression: Write a function that takes a string as input and compresses it by replacing consecutive repeated characters with the character followed by the count of consecutive occurrences. For example, the string "aabcccccaaa" would be compressed to "a2b1c5a3".
//
// Validate Parentheses: Write a function that takes a string containing parentheses, brackets, and braces as input and returns true if the parentheses are properly nested and balanced, and false otherwise. For example, the string "{[()()]}" would return true, while "([)]" would return false.
//
// Binary Search: Write a function that implements the binary search algorithm to search for a target value in a sorted array. The function should return the index of the target value if it exists in the array, or -1 if it does not.
//
// Merge Intervals: Given a collection of intervals (represented as pairs of integers), write a function that merges overlapping intervals and returns a new collection of non-overlapping intervals. For example, given the input [(1, 3), (2, 6), (8, 10), (15, 18)], the function should return [(1, 6), (8, 10), (15, 18)].
//
// Counting Elements: Given an array of integers, write a function that counts the number of elements that are smaller than the current element to the right of it. For example, given the input [3, 4, 9, 6, 1], the function should return [2, 2, 1, 1, 0].

func main() {
	fmt.Println(isPalindrome("racecar"))
	fmt.Println(isPalindrome("921129"))
	fmt.Println(isPalindrome("92129"))
	fmt.Println(isPalindrome("ricardo"))
	fmt.Println(isPalindrome("racecars"))
}
