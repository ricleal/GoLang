package main

import (
	"fmt"
	"testing"
)

func BenchmarkV1(b *testing.B) {

	for i := 0; i < b.N; i++ {
		v1()
	}
}

func BenchmarkV2(b *testing.B) {

	for i := 0; i < b.N; i++ {
		v2()
	}
}

func BenchmarkDownloadFile(b *testing.B) {

	for i := 0; i < b.N; i++ {
		filename := fmt.Sprintf("/tmp/file-%d.txt", i)
		downloadFile(filename)
	}
}
