package vad

import (
	"testing"
)

func TestGaussianProbability(t *testing.T) {
	var delta int16

	// Input value at mean.
	if got := gaussianProbability(0, 0, 128, &delta); got != 1048576 {
		t.Errorf("gaussianProbability(0, 0, 128) = %d, want 1048576", got)
	}
	if delta != 0 {
		t.Errorf("delta = %d, want 0", delta)
	}

	if got := gaussianProbability(16, 128, 128, &delta); got != 1048576 {
		t.Errorf("gaussianProbability(16, 128, 128) = %d, want 1048576", got)
	}
	if delta != 0 {
		t.Errorf("delta = %d, want 0", delta)
	}

	if got := gaussianProbability(-16, -128, 128, &delta); got != 1048576 {
		t.Errorf("gaussianProbability(-16, -128, 128) = %d, want 1048576", got)
	}
	if delta != 0 {
		t.Errorf("delta = %d, want 0", delta)
	}

	// Largest possible input to give non-zero probability.
	if got := gaussianProbability(59, 0, 128, &delta); got != 1024 {
		t.Errorf("gaussianProbability(59, 0, 128) = %d, want 1024", got)
	}
	if delta != 7552 {
		t.Errorf("delta = %d, want 7552", delta)
	}

	if got := gaussianProbability(75, 128, 128, &delta); got != 1024 {
		t.Errorf("gaussianProbability(75, 128, 128) = %d, want 1024", got)
	}
	if delta != 7552 {
		t.Errorf("delta = %d, want 7552", delta)
	}

	if got := gaussianProbability(-75, -128, 128, &delta); got != 1024 {
		t.Errorf("gaussianProbability(-75, -128, 128) = %d, want 1024", got)
	}
	if delta != -7552 {
		t.Errorf("delta = %d, want -7552", delta)
	}

	// Too large input, should give zero probability.
	if got := gaussianProbability(105, 0, 128, &delta); got != 0 {
		t.Errorf("gaussianProbability(105, 0, 128) = %d, want 0", got)
	}
	if delta != 13440 {
		t.Errorf("delta = %d, want 13440", delta)
	}
}
