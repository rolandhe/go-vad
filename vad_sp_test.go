package vad

import (
	"testing"
)

const kMaxFrameLenSp = 960

func TestDownsampling(t *testing.T) {
	zeros := make([]int16, kMaxFrameLenSp)
	dataOut := make([]int16, kMaxFrameLenSp)
	state := make([]int32, 2)

	// Input all zeros.
	downsampling(zeros, dataOut, state)
	if state[0] != 0 {
		t.Errorf("state[0] = %d, want 0", state[0])
	}
	if state[1] != 0 {
		t.Errorf("state[1] = %d, want 0", state[1])
	}
	for i := 0; i < kMaxFrameLenSp/2; i++ {
		if dataOut[i] != 0 {
			t.Errorf("dataOut[%d] = %d, want 0", i, dataOut[i])
		}
	}

	// Non-zero test with i*i signal.
	dataIn := make([]int16, kMaxFrameLenSp)
	for i := 0; i < kMaxFrameLenSp; i++ {
		// i*i will wrap around in int16, matching C behavior.
		dataIn[i] = int16(i * i)
	}
	state = make([]int32, 2)
	downsampling(dataIn, dataOut, state)
	if state[0] != 207 {
		t.Errorf("state[0] = %d, want 207", state[0])
	}
	if state[1] != 2270 {
		t.Errorf("state[1] = %d, want 2270", state[1])
	}
}

func TestFindMinimum(t *testing.T) {
	self := &vadInstT{}
	if initCore(self) != 0 {
		t.Fatal("initCore failed")
	}

	kReferenceMin := [32]int16{
		1600, 720, 509, 512, 532, 552, 570, 588,
		606, 624, 642, 659, 675, 691, 707, 723,
		1600, 544, 502, 522, 542, 561, 579, 597,
		615, 633, 651, 667, 683, 699, 715, 731,
	}

	for i := int16(0); i < 16; i++ {
		value := 500 * (i + 1)
		for j := 0; j < kNumChannels; j++ {
			got := findMinimum(self, value, j)
			want := kReferenceMin[i]
			if got != want {
				t.Errorf("FindMinimum(self, %d, %d) iter %d = %d, want %d",
					value, j, i, got, want)
			}
			got2 := findMinimum(self, 12000, j)
			want2 := kReferenceMin[i+16]
			if got2 != want2 {
				t.Errorf("FindMinimum(self, 12000, %d) iter %d = %d, want %d",
					j, i, got2, want2)
			}
		}
		self.frameCounter++
	}
}
