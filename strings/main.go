package main

import (
	"fmt"
)

func lengthOfLongestSubstring(s string) int {
	var max int
	var start int
	var end int
	var m = make(map[byte]int)
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

func main() {
	fmt.Println(lengthOfLongestSubstring("aab"))
}
