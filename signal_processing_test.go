package vad

import (
	"testing"
)

// countLeadingZeros32Table is the lookup table for countLeadingZeros32NotBuiltin.
var countLeadingZeros32Table = [64]int8{
	32, 8, 17, -1, -1, 14, -1, -1, -1, 20, -1, -1, -1, 28, -1, 18,
	-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 0, 26, 25, 24,
	4, 11, 23, 31, 3, 7, 10, 16, 22, 30, -1, -1, 2, 6, 13, 9,
	-1, 15, -1, 21, -1, 29, 19, -1, -1, -1, -1, -1, 1, 27, 5, 12,
}

// countLeadingZeros32NotBuiltin is the table-based fallback for testing.
func countLeadingZeros32NotBuiltin(n uint32) int {
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return int(countLeadingZeros32Table[(n*0x8c0b2891)>>26])
}

func TestCountLeadingZeros32(t *testing.T) {
	// Test zero.
	if got := countLeadingZeros32(0); got != 32 {
		t.Errorf("countLeadingZeros32(0) = %d, want 32", got)
	}
	if got := countLeadingZeros32NotBuiltin(0); got != 32 {
		t.Errorf("countLeadingZeros32NotBuiltin(0) = %d, want 32", got)
	}
	// Test all bit positions.
	for i := 0; i < 32; i++ {
		singleOne := uint32(1) << i
		allOnes := 2*singleOne - 1
		want := 31 - i
		if got := countLeadingZeros32(singleOne); got != want {
			t.Errorf("countLeadingZeros32(1<<%d) = %d, want %d", i, got, want)
		}
		if got := countLeadingZeros32NotBuiltin(singleOne); got != want {
			t.Errorf("countLeadingZeros32NotBuiltin(1<<%d) = %d, want %d", i, got, want)
		}
		if got := countLeadingZeros32(allOnes); got != want {
			t.Errorf("countLeadingZeros32(allOnes for bit %d) = %d, want %d", i, got, want)
		}
		if got := countLeadingZeros32NotBuiltin(allOnes); got != want {
			t.Errorf("countLeadingZeros32NotBuiltin(allOnes for bit %d) = %d, want %d", i, got, want)
		}
	}
}

func TestGetSizeInBits(t *testing.T) {
	a32 := int32(111121)
	want := int16(17)
	if got := getSizeInBits(uint32(a32)); got != want {
		t.Errorf("getSizeInBits(%d) = %d, want %d", a32, got, want)
	}
}

func TestNormW32(t *testing.T) {
	if got := normW32(0); got != 0 {
		t.Errorf("normW32(0) = %d, want 0", got)
	}
	if got := normW32(-1); got != 31 {
		t.Errorf("normW32(-1) = %d, want 31", got)
	}
	if got := normW32(word32Min); got != 0 {
		t.Errorf("normW32(MIN) = %d, want 0", got)
	}
	a32 := int32(111121)
	if got := normW32(a32); got != 14 {
		t.Errorf("normW32(%d) = %d, want 14", a32, got)
	}
}

func TestNormU32(t *testing.T) {
	if got := normU32(0); got != 0 {
		t.Errorf("normU32(0) = %d, want 0", got)
	}
	if got := normU32(0xffffffff); got != 0 {
		t.Errorf("normU32(0xFFFFFFFF) = %d, want 0", got)
	}
	a32 := int32(111121)
	if got := normU32(uint32(a32)); got != 15 {
		t.Errorf("normU32(%d) = %d, want 15", a32, got)
	}
}

func TestDivW32W16(t *testing.T) {
	got := divW32W16(117, -5)
	want := int32(-23)
	if got != want {
		t.Errorf("divW32W16(117, -5) = %d, want %d", got, want)
	}
}

func TestSPLMul(t *testing.T) {
	a, b := -3, int16(21)
	// In C: WEBRTC_SPL_MUL(a, B) = (int32_t)((int32_t)(a) * (int32_t)(B))
	got := int32(a) * int32(b)
	want := int32(-63)
	if got != want {
		t.Errorf("int32(%d)*int32(%d) = %d, want %d", a, b, got, want)
	}
	// Overflow case.
	c := int16(-3)
	d := int32(word32Max)
	got2 := int32(c) * d
	want2 := int32(-2147483645)
	if got2 != want2 {
		t.Errorf("int32(%d)*%d = %d, want %d", c, d, got2, want2)
	}
}

func TestEnergy(t *testing.T) {
	A := []int16{1, 2, 33, 100}
	bScale := 0
	got := energy(A, &bScale)
	want := int32(11094)
	if got != want {
		t.Errorf("energy({1,2,33,100}) = %d, want %d", got, want)
	}
	if bScale != 0 {
		t.Errorf("energy scale = %d, want 0", bScale)
	}
}

func TestResample48khzTo32khz(t *testing.T) {
	const kBlockSize = 16

	// Saturated input vector of 48 samples + 7 extra (like C test).
	kVectorSaturated := []int32{
		-32768, -32768, -32768, -32768, -32768, -32768, -32768, -32768,
		-32768, -32768, -32768, -32768, -32768, -32768, -32768, -32768,
		-32768, -32768, -32768, -32768, -32768, -32768, -32768, -32768,
		32767, 32767, 32767, 32767, 32767, 32767, 32767, 32767,
		32767, 32767, 32767, 32767, 32767, 32767, 32767, 32767,
		32767, 32767, 32767, 32767, 32767, 32767, 32767, 32767,
		32767, 32767, 32767, 32767, 32767, 32767, 32767,
	}

	const kRefValue32kHz1 = -1077493760
	const kRefValue32kHz2 = 1077493645

	outVector := make([]int32, 2*kBlockSize)
	resample48khzTo32khz(kVectorSaturated, outVector, kBlockSize)

	// Comparing output values against references.
	// Values at position 12-15 are skipped to account for the filter lag.
	for i := 0; i < 12; i++ {
		if outVector[i] != kRefValue32kHz1 {
			t.Errorf("out_vector[%d] = %d, want %d", i, outVector[i], kRefValue32kHz1)
		}
	}
	for i := 16; i < 2*kBlockSize; i++ {
		if outVector[i] != kRefValue32kHz2 {
			t.Errorf("out_vector[%d] = %d, want %d", i, outVector[i], kRefValue32kHz2)
		}
	}
}
