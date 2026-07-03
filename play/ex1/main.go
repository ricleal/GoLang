package main

import (
	"errors"
	"log"
	"math/rand"
	"sync"
)

func makeFrame(width, height int) [][]int {
	frame := make([][]int, height)
	for i := range frame {
		frame[i] = make([]int, width)
		// populate the row with random values
		for j := range frame[i] {
			frame[i][j] = rand.Intn(256) // example: random value between 0-255
		}
	}
	return frame
}

func frameValue(frame [][]int) (float64, error) {
	var v int
	for i := range frame {
		for j := range frame[i] {
			v += frame[i][j]
		}
	}
	avergeValue := float64(v) / float64(len(frame)*len(frame[0]))
	if avergeValue < 127.5 {
		return 0, errors.New("average value below threshold")
	}
	return avergeValue, nil
}

const (
	frameWidth  = 1920
	frameHeight = 1080
	nFrames     = 60 // number of jobs
	nWorkers    = 4
)

func worker(id int, frames <-chan [][]int, results chan<- int, errors chan<- error) {
	log.Printf("Worker: %d", id)
	for frame := range frames {
		value, err := frameValue(frame)
		if err != nil {
			errors <- err
			continue
		}
		results <- int(value)
	}
}

func main() {
	frames := make(chan [][]int, nFrames)
	results := make(chan int)
	errors := make(chan error)

	// start workers
	var wgWorkers sync.WaitGroup
	for w := 1; w <= nWorkers; w++ {
		wgWorkers.Go(func() {
			worker(w, frames, results, errors)
		})
	}

	// send frames to workers asynchronously
	wgFrames := sync.WaitGroup{}
	wgFrames.Go(func() {
		for i := 0; i < nFrames; i++ {
			frames <- makeFrame(frameWidth, frameHeight)
		}
		close(frames)
	})

	// collect results and errors asynchronously
	var wgResults sync.WaitGroup
	wgResults.Go(func() {
		nErrors := 0
		total := 0
		for {
			select {
			case result, ok := <-results:
				if !ok {
					results = nil
				} else {
					log.Printf("Result: %d", result)
					total += 1
				}
			case err, ok := <-errors:
				if !ok {
					errors = nil
				} else {
					log.Printf("Error: %v", err)
					nErrors++
					total += 1
				}
			}
			if results == nil && errors == nil {
				log.Printf("Total Errors: %d out of %d", nErrors, total)
				break
			}
		}
	})

	wgFrames.Wait()
	wgWorkers.Wait()
	close(results)
	close(errors)
	wgResults.Wait()
}
