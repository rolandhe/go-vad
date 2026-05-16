package vad

import (
	"testing"
)

func TestCalculateFeatures(t *testing.T) {
	const kNumValidFrameLengths = 3
	kReference := [kNumValidFrameLengths]int16{48, 11, 11}
	kFeatures := [kNumValidFrameLengths * kNumChannels]int16{
		1213, 759, 587, 462, 434, 272,
		1479, 1385, 1291, 1200, 1103, 1099,
		1732, 1692, 1681, 1629, 1436, 1436,
	}
	kOffset := [6]int16{368, 368, 272, 176, 176, 176}

	// Construct a speech signal that will trigger the VAD in all modes.
	kMaxFrameLen := 1440
	speech := make([]int16, kMaxFrameLen)
	for i := 0; i < kMaxFrameLen; i++ {
		speech[i] = int16(i * i)
	}
	features := make([]int16, kNumChannels)

	// Test valid frame lengths (kRates[0] = 8000).
	kRates := []int{8000, 12000, 16000, 24000, 32000, 48000}
	kFrameLengths := []int{80, 120, 160, 240, 320, 480, 640, 960, 1440}

	// Share state across frame lengths like the C test.
	speechSelf := &vadInstT{}
	if initCore(speechSelf) != 0 {
		t.Fatal("initCore failed")
	}
	frameLengthIndex := 0
	for _, fl := range kFrameLengths {
		if !isValidRateAndFrameLen(kRates[0], fl) {
			continue
		}
		got := calculateFeatures(speechSelf, speech, fl, features)
		want := kReference[frameLengthIndex]
		if got != want {
			t.Errorf("CalculateFeatures(speech, %d) totalEnergy = %d, want %d", fl, got, want)
		}
		for k := 0; k < kNumChannels; k++ {
			wantFeat := kFeatures[k+frameLengthIndex*kNumChannels]
			if features[k] != wantFeat {
				t.Errorf("features[%d] at len=%d = %d, want %d", k, fl, features[k], wantFeat)
			}
		}
		frameLengthIndex++
	}
	if frameLengthIndex != kNumValidFrameLengths {
		t.Errorf("frameLengthIndex = %d, want %d", frameLengthIndex, kNumValidFrameLengths)
	}

	// All zeros. C test resets state for this test.
	zeros := make([]int16, kMaxFrameLen)
	zerosSelf := &vadInstT{}
	if initCore(zerosSelf) != 0 {
		t.Fatal("initCore failed")
	}
	for _, fl := range kFrameLengths {
		if !isValidRateAndFrameLen(kRates[0], fl) {
			continue
		}
		got := calculateFeatures(zerosSelf, zeros, fl, features)
		if got != 0 {
			t.Errorf("CalculateFeatures(zeros, %d) = %d, want 0", fl, got)
		}
		for k := 0; k < kNumChannels; k++ {
			if features[k] != kOffset[k] {
				t.Errorf("features[%d] (zeros, len=%d) = %d, want %d", k, fl, features[k], kOffset[k])
			}
		}
	}

	// All ones. C test resets state for this test.
	ones := make([]int16, kMaxFrameLen)
	for i := 0; i < kMaxFrameLen; i++ {
		ones[i] = 1
	}
	onesSelf := &vadInstT{}
	if initCore(onesSelf) != 0 {
		t.Fatal("initCore failed")
	}
	for _, fl := range kFrameLengths {
		if !isValidRateAndFrameLen(kRates[0], fl) {
			continue
		}
		got := calculateFeatures(onesSelf, ones, fl, features)
		if got != 0 {
			t.Errorf("CalculateFeatures(ones, %d) = %d, want 0", fl, got)
		}
		for k := 0; k < kNumChannels; k++ {
			if features[k] != kOffset[k] {
				t.Errorf("features[%d] (ones, len=%d) = %d, want %d", k, fl, features[k], kOffset[k])
			}
		}
	}
}

// isValidRateAndFrameLen checks if the rate and frame length combination is valid.
func isValidRateAndFrameLen(rate int, frameLength int) bool {
	switch rate {
	case 8000:
		return frameLength == 80 || frameLength == 160 || frameLength == 240
	case 16000:
		return frameLength == 160 || frameLength == 320 || frameLength == 480
	case 32000:
		return frameLength == 320 || frameLength == 640 || frameLength == 960
	case 48000:
		return frameLength == 480 || frameLength == 960 || frameLength == 1440
	}
	return false
}
