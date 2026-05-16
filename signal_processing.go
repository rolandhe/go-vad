// Copyright (c) 2012 The WebRTC project authors. All Rights Reserved.
// Copyright (c) 2016 Daniel Pirch.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package vad

import "math/bits"

// WebRTC fixed-point signal processing utility functions.

const (
	word16Max = 32767
	word16Min = -32768
	word32Max = 0x7fffffff
	word32Min = -0x80000000
)

// countLeadingZeros32 returns the number of leading zero bits.
func countLeadingZeros32(n uint32) int {
	if n == 0 {
		return 32
	}
	return bits.LeadingZeros32(n)
}

// getSizeInBits returns the minimum number of bits to represent n.
func getSizeInBits(n uint32) int16 {
	return int16(32 - countLeadingZeros32(n))
}

// normW32 returns the number of steps a can be left-shifted without overflow,
// or 0 if a == 0.
func normW32(a int32) int16 {
	if a == 0 {
		return 0
	}
	if a < 0 {
		a = ^a
	}
	return int16(countLeadingZeros32(uint32(a)) - 1)
}

// normU32 returns the number of steps a can be left-shifted without overflow,
// or 0 if a == 0.
func normU32(a uint32) int16 {
	if a == 0 {
		return 0
	}
	return int16(countLeadingZeros32(a))
}

// divW32W16 divides an int32 by int16. Returns 0x7FFFFFFF on division by zero.
func divW32W16(num int32, den int16) int32 {
	if den != 0 {
		return num / int32(den)
	}
	return 0x7FFFFFFF
}

// getScalingSquare determines the number of right-shifts needed to avoid
// overflow when squaring and summing a vector.
func getScalingSquare(inVector []int16, times int) int16 {
	nbits := getSizeInBits(uint32(times))
	var smax int16 = -1
	for _, v := range inVector {
		var sabs int16
		if v > 0 {
			sabs = v
		} else {
			sabs = -v
		}
		if sabs > smax {
			smax = sabs
		}
	}
	t := normW32(int32(smax) * int32(smax))
	if smax == 0 {
		return 0
	}
	if t > nbits {
		return 0
	}
	return nbits - t
}

// energy computes sum of squares of a vector with scaling to avoid overflow.
func energy(vector []int16, scaleFactor *int) int32 {
	scaling := getScalingSquare(vector, len(vector))
	var en int32
	for _, v := range vector {
		en += (int32(v) * int32(v)) >> scaling
	}
	*scaleFactor = int(scaling)
	return en
}
