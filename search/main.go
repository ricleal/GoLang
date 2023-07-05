package main

//TODO: implement a binary search tree

type Data struct {
	data []string
}

func (d *Data) Setup(data []string) {
	// sort the input data
	d.data = data
	// quick sort Divide and Conquer
	// worst-case time complexity of O(n log n) and a best-case time complexity of O(n).

}

func quickSort(data []string) {

}

func (d *Data) IsIn(key string) bool {
	// Binary search: Time complexity is O(log n)
	return false
}
