package main

import (
	"testing"

	"golang.org/x/exp/rand"
)

const (
	providersCount = 2 ^ 100
	dataLength     = 2 ^ 10000
)

type LargeStruct struct {
	Data  []float64
	Index int
}

var providers = make([]LargeStruct, providersCount)

func initDataLength() {
	for i := 0; i < providersCount; i++ {
		providers[i].Data = make([]float64, dataLength)
	}
}

func addRandomDataToLargeStruct(s *LargeStruct) {
	for i := 0; i < dataLength; i++ {
		// random number between 0 and dataLength
		randomNumber := rand.Intn(dataLength)
		s.Data[randomNumber] = rand.Float64()
	}
}

////

func BenchmarkStructByValue(b *testing.B) {
	initDataLength()
	for i := 0; i < b.N; i++ {
		var newProviders []LargeStruct
		for j, p := range providers {
			addRandomDataToLargeStruct(&p)
			p.Index = j
			newProviders = append(newProviders, p)
		}
		_ = newProviders
	}
}

func BenchmarkStructForByReference(b *testing.B) {
	initDataLength()
	for i := 0; i < b.N; i++ {
		var newProviders []LargeStruct
		for j := 0; j < providersCount; j++ {
			p := providers[j]
			addRandomDataToLargeStruct(&p)
			p.Index = j
			newProviders = append(newProviders, p)
		}
		_ = newProviders
	}
}

func BenchmarkStructForByPointer(b *testing.B) {
	initDataLength()
	for i := 0; i < b.N; i++ {
		var newProviders []LargeStruct
		for j := 0; j < providersCount; j++ {
			p := &providers[j]
			addRandomDataToLargeStruct(p)
			p.Index = j
			newProviders = append(newProviders, *p)
		}
		_ = newProviders
	}
}
