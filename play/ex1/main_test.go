package main

import "testing"

func TestMakeFrameDimensionsAndRange(t *testing.T) {
	width, height := 5, 3
	frame := makeFrame(width, height)

	if len(frame) != height {
		t.Fatalf("expected %d rows, got %d", height, len(frame))
	}

	for i := range frame {
		if len(frame[i]) != width {
			t.Fatalf("row %d: expected %d cols, got %d", i, width, len(frame[i]))
		}
		for j := range frame[i] {
			if frame[i][j] < 0 || frame[i][j] > 255 {
				t.Fatalf("cell (%d,%d): value %d out of [0,255]", i, j, frame[i][j])
			}
		}
	}
}

func TestFrameValueThreshold(t *testing.T) {
	belowThreshold := [][]int{{127, 127}, {127, 127}}
	if _, err := frameValue(belowThreshold); err == nil {
		t.Fatal("expected error for average below threshold")
	}

	aboveThreshold := [][]int{{128, 128}, {128, 128}}
	avg, err := frameValue(aboveThreshold)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if avg != 128 {
		t.Fatalf("expected average 128, got %v", avg)
	}
}

func TestWorkerRoutesResultsAndErrors(t *testing.T) {
	frames := make(chan [][]int, 2)
	results := make(chan int, 2)
	errs := make(chan error, 2)

	frames <- [][]int{{128, 128}, {128, 128}}
	frames <- [][]int{{0, 0}, {0, 0}}
	close(frames)

	worker(1, frames, results, errs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}

	got := <-results
	if got != 128 {
		t.Fatalf("expected result 128, got %d", got)
	}
}
