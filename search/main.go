package main

import (
	"fmt"
	"sort"

	"golang.org/x/exp/constraints"
)

// Generic Binary Search Tree implementation
type TreeNode[T constraints.Ordered] struct {
	Value T
	Left  *TreeNode[T]
	Right *TreeNode[T]
}

type BST[T constraints.Ordered] struct {
	Root *TreeNode[T]
}

// Insert adds a value to the BST
func (bst *BST[T]) Insert(value T) {
	bst.Root = insertNode(bst.Root, value)
}

func insertNode[T constraints.Ordered](node *TreeNode[T], value T) *TreeNode[T] {
	if node == nil {
		return &TreeNode[T]{Value: value}
	}

	if value < node.Value {
		node.Left = insertNode(node.Left, value)
	} else if value > node.Value {
		node.Right = insertNode(node.Right, value)
	}
	// If equal, don't insert duplicate

	return node
}

// Search finds a value in the BST
func (bst *BST[T]) Search(value T) bool {
	return searchNode(bst.Root, value)
}

func searchNode[T constraints.Ordered](node *TreeNode[T], value T) bool {
	if node == nil {
		return false
	}

	if value == node.Value {
		return true
	} else if value < node.Value {
		return searchNode(node.Left, value)
	} else {
		return searchNode(node.Right, value)
	}
}

// InOrder traversal (Left -> Root -> Right) - returns sorted order
func (bst *BST[T]) InOrder() []T {
	var result []T
	inOrderTraversal(bst.Root, &result)
	return result
}

func inOrderTraversal[T constraints.Ordered](node *TreeNode[T], result *[]T) {
	if node != nil {
		inOrderTraversal(node.Left, result)
		*result = append(*result, node.Value)
		inOrderTraversal(node.Right, result)
	}
}

// Generic Data struct with quicksort and binary search
type Data[T constraints.Ordered] struct {
	data []T
}

// Setup sorts the input data using quicksort
func (d *Data[T]) Setup(data []T) {
	d.data = make([]T, len(data))
	copy(d.data, data)
	d.data = quickSort(d.data)
}

// quickSort implements the QuickSort algorithm with generics
// Time Complexity: Average O(n log n), Worst O(n²)
// Space Complexity: O(log n) for recursion stack
func quickSort[T constraints.Ordered](data []T) []T {
	if len(data) <= 1 {
		return data
	}

	// Choose pivot (middle element)
	pivot := data[len(data)/2]

	var left, middle, right []T

	// Partition
	for _, v := range data {
		if v < pivot {
			left = append(left, v)
		} else if v == pivot {
			middle = append(middle, v)
		} else {
			right = append(right, v)
		}
	}

	// Recursively sort and combine
	return append(append(quickSort(left), middle...), quickSort(right)...)
}

// IsIn performs binary search on sorted data
// Time Complexity: O(log n)
func (d *Data[T]) IsIn(key T) bool {
	left, right := 0, len(d.data)-1

	for left <= right {
		mid := left + (right-left)/2

		if d.data[mid] == key {
			return true
		} else if d.data[mid] < key {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return false
}

func (d *Data[T]) GetData() []T {
	return d.data
}

func main() {
	fmt.Println("=== GENERIC Binary Search Tree Demo ===\n")

	// BST with strings
	fmt.Println("--- BST with Strings ---")
	bstStrings := &BST[string]{}
	words := []string{"dog", "cat", "fish", "bird", "ant", "elephant", "zebra"}

	fmt.Println("Inserting words:", words)
	for _, word := range words {
		bstStrings.Insert(word)
	}

	fmt.Println("BST In-Order (sorted):", bstStrings.InOrder())

	searchTerms := []string{"cat", "dog", "lion", "zebra"}
	for _, term := range searchTerms {
		found := bstStrings.Search(term)
		status := "❌"
		if found {
			status = "✅"
		}
		fmt.Printf("  Search '%s': %s\n", term, status)
	}

	// BST with integers
	fmt.Println("\n--- BST with Integers ---")
	bstInts := &BST[int]{}
	numbers := []int{50, 30, 70, 20, 40, 60, 80}

	fmt.Println("Inserting numbers:", numbers)
	for _, num := range numbers {
		bstInts.Insert(num)
	}

	fmt.Println("BST In-Order (sorted):", bstInts.InOrder())

	searchNums := []int{20, 35, 60, 100}
	for _, num := range searchNums {
		found := bstInts.Search(num)
		status := "❌"
		if found {
			status = "✅"
		}
		fmt.Printf("  Search %d: %s\n", num, status)
	}

	// Generic QuickSort + Binary Search with strings
	fmt.Println("\n\n=== GENERIC QuickSort + Binary Search ===\n")

	fmt.Println("--- With Strings ---")
	dataStrings := &Data[string]{}
	unsorted := []string{"grape", "apple", "cherry", "banana", "date", "fig", "elderberry"}

	fmt.Println("Original:", unsorted)
	dataStrings.Setup(unsorted)
	fmt.Println("Sorted:  ", dataStrings.GetData())

	for _, term := range searchTerms {
		found := dataStrings.IsIn(term)
		status := "❌"
		if found {
			status = "✅"
		}
		fmt.Printf("  Search '%s': %s\n", term, status)
	}

	// Generic QuickSort + Binary Search with integers
	fmt.Println("\n--- With Integers ---")
	dataInts := &Data[int]{}
	unsortedInts := []int{64, 34, 25, 12, 22, 11, 90}

	fmt.Println("Original:", unsortedInts)
	dataInts.Setup(unsortedInts)
	fmt.Println("Sorted:  ", dataInts.GetData())

	for _, num := range searchNums {
		found := dataInts.IsIn(num)
		status := "❌"
		if found {
			status = "✅"
		}
		fmt.Printf("  Search %d: %s\n", num, status)
	}

	// Generic QuickSort + Binary Search with floats
	fmt.Println("\n--- With Floats ---")
	dataFloats := &Data[float64]{}
	unsortedFloats := []float64{3.14, 2.71, 1.41, 1.73, 2.23}

	fmt.Println("Original:", unsortedFloats)
	dataFloats.Setup(unsortedFloats)
	fmt.Println("Sorted:  ", dataFloats.GetData())

	searchFloats := []float64{1.41, 2.71, 9.99}
	for _, num := range searchFloats {
		found := dataFloats.IsIn(num)
		status := "❌"
		if found {
			status = "✅"
		}
		fmt.Printf("  Search %.2f: %s\n", num, status)
	}

	// Comparison
	fmt.Println("\n\n=== Generics Benefits ===")
	fmt.Println("┌────────────────────────────────────────────────────────┐")
	fmt.Println("│ ✅ Type Safety: Compile-time type checking             │")
	fmt.Println("│ ✅ Code Reuse: One implementation for all types        │")
	fmt.Println("│ ✅ Performance: No interface{} boxing overhead          │")
	fmt.Println("│ ✅ Works with: int, string, float64, any ordered type  │")
	fmt.Println("└────────────────────────────────────────────────────────┘")

	// Bonus: Using Go's standard library
	fmt.Println("\n--- Go stdlib comparison ---")
	fruits := []string{"grape", "apple", "cherry", "banana", "date"}
	sort.Strings(fruits)
	fmt.Println("Sorted:", fruits)

	index := sort.SearchStrings(fruits, "cherry")
	if index < len(fruits) && fruits[index] == "cherry" {
		fmt.Printf("Found 'cherry' at index %d\n", index)
	}
}
