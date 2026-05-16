package vad

import (
	"testing"
)

var (
	kModes            = []int{0, 1, 2, 3}
	kModesSize        = len(kModes)
	kRates            = []int{8000, 12000, 16000, 24000, 32000, 48000}
	kRatesSize        = len(kRates)
	kFrameLengths     = []int{80, 120, 160, 240, 320, 480, 640, 960, 1440}
	kFrameLengthsSize = len(kFrameLengths)
	kMaxFrameLength   = 1440
)

func TestInitCore(t *testing.T) {
	// Null pointer test - Go doesn't have null pointers for structs, but we test initFlag.
	self := &vadInstT{}

	// Verify return = 0 for non-null pointer.
	if got := initCore(self); got != 0 {
		t.Errorf("initCore = %d, want 0", got)
	}
	// Verify init_flag is set.
	if self.initFlag != 42 {
		t.Errorf("initFlag = %d, want 42", self.initFlag)
	}
}

func TestSetModeCore(t *testing.T) {
	self := &vadInstT{}
	if got := initCore(self); got != 0 {
		t.Fatal("initCore failed")
	}

	// Invalid modes should return -1.
	if got := setModeCore(self, -1); got != -1 {
		t.Errorf("setModeCore(-1) = %d, want -1", got)
	}
	if got := setModeCore(self, 1000); got != -1 {
		t.Errorf("setModeCore(1000) = %d, want -1", got)
	}

	// Valid modes should return 0.
	for _, mode := range kModes {
		if got := setModeCore(self, mode); got != 0 {
			t.Errorf("setModeCore(%d) = %d, want 0", mode, got)
		}
	}
}

func TestCalcVadZeros(t *testing.T) {
	zeros := make([]int16, kMaxFrameLength)

	for _, fl := range kFrameLengths {
		if isValidRateAndFrameLen(8000, fl) {
			inst := &vadInstT{}
			if initCore(inst) != 0 {
				t.Fatal("initCore failed")
			}
			if got := calcVad8khz(inst, zeros[:fl], fl); int16(got) != 0 {
				t.Errorf("calcVad8khz(zeros, %d) = %d, want 0", fl, got)
			}
		}
		if isValidRateAndFrameLen(16000, fl) {
			inst := &vadInstT{}
			if initCore(inst) != 0 {
				t.Fatal("initCore failed")
			}
			if got := calcVad16khz(inst, zeros[:fl], fl); int16(got) != 0 {
				t.Errorf("calcVad16khz(zeros, %d) = %d, want 0", fl, got)
			}
		}
		if isValidRateAndFrameLen(32000, fl) {
			inst := &vadInstT{}
			if initCore(inst) != 0 {
				t.Fatal("initCore failed")
			}
			if got := calcVad32khz(inst, zeros[:fl], fl); int16(got) != 0 {
				t.Errorf("calcVad32khz(zeros, %d) = %d, want 0", fl, got)
			}
		}
		if isValidRateAndFrameLen(48000, fl) {
			inst := &vadInstT{}
			if initCore(inst) != 0 {
				t.Fatal("initCore failed")
			}
			if got := calcVad48khz(inst, zeros[:fl], fl); int16(got) != 0 {
				t.Errorf("calcVad48khz(zeros, %d) = %d, want 0", fl, got)
			}
		}
	}
}

func TestCalcVadSpeech(t *testing.T) {
	speech := make([]int16, kMaxFrameLength)
	for i := 0; i < kMaxFrameLength; i++ {
		speech[i] = int16(i * i)
	}

	for _, fl := range kFrameLengths {
		if isValidRateAndFrameLen(8000, fl) {
			inst := &vadInstT{}
			if initCore(inst) != 0 {
				t.Fatal("initCore failed")
			}
			if got := calcVad8khz(inst, speech[:fl], fl); int16(got) != 1 {
				t.Errorf("calcVad8khz(speech, %d) = %d, want 1", fl, got)
			}
		}
		if isValidRateAndFrameLen(16000, fl) {
			inst := &vadInstT{}
			if initCore(inst) != 0 {
				t.Fatal("initCore failed")
			}
			if got := calcVad16khz(inst, speech[:fl], fl); int16(got) != 1 {
				t.Errorf("calcVad16khz(speech, %d) = %d, want 1", fl, got)
			}
		}
		if isValidRateAndFrameLen(32000, fl) {
			inst := &vadInstT{}
			if initCore(inst) != 0 {
				t.Fatal("initCore failed")
			}
			if got := calcVad32khz(inst, speech[:fl], fl); int16(got) != 1 {
				t.Errorf("calcVad32khz(speech, %d) = %d, want 1", fl, got)
			}
		}
		if isValidRateAndFrameLen(48000, fl) {
			inst := &vadInstT{}
			if initCore(inst) != 0 {
				t.Fatal("initCore failed")
			}
			if got := calcVad48khz(inst, speech[:fl], fl); int16(got) != 1 {
				t.Errorf("calcVad48khz(speech, %d) = %d, want 1", fl, got)
			}
		}
	}
}
