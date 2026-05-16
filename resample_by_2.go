// Copyright (c) 2011 The WebRTC project authors. All Rights Reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package vad

// Allpass filter coefficients.
// Read-only at runtime.
var kResampleAllpass = [2][3]int16{
	{821, 6110, 12382},
	{3050, 9368, 15063},
}

// downBy2IntToShort decimates int32 input to int16 output by factor 2.
// Input:  int32 (shifted 15 positions to the left, + offset 16384) OVERWRITTEN!
// Output: int16 (saturated) (of length len/2)
// State:  filter state array; length = 8
func downBy2IntToShort(in []int32, out []int16, state []int32) {
	length := len(in) >> 1

	// lower allpass filter (operates on even input samples)
	for i := 0; i < length; i++ {
		tmp0 := in[i<<1]
		// overflow: matches C RTC_NO_SANITIZE("signed-integer-overflow") in resample_by_2_internal.c
		diff := tmp0 - state[1]
		diff = (diff + (1 << 13)) >> 14
		tmp1 := state[0] + diff*int32(kResampleAllpass[1][0])
		state[0] = tmp0
		diff = tmp1 - state[2]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		tmp0 = state[1] + diff*int32(kResampleAllpass[1][1])
		state[1] = tmp1
		diff = tmp0 - state[3]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		state[3] = state[2] + diff*int32(kResampleAllpass[1][2])
		state[2] = tmp0

		in[i<<1] = state[3] >> 1
	}

	// upper allpass filter (operates on odd input samples)
	for i := 0; i < length; i++ {
		tmp0 := in[i<<1+1]
		// overflow: matches C RTC_NO_SANITIZE("signed-integer-overflow")
		diff := tmp0 - state[5]
		diff = (diff + (1 << 13)) >> 14
		tmp1 := state[4] + diff*int32(kResampleAllpass[0][0])
		state[4] = tmp0
		diff = tmp1 - state[6]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		tmp0 = state[5] + diff*int32(kResampleAllpass[0][1])
		state[5] = tmp1
		diff = tmp0 - state[7]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		state[7] = state[6] + diff*int32(kResampleAllpass[0][2])
		state[6] = tmp0

		in[i<<1+1] = state[7] >> 1
	}

	// combine allpass outputs
	for i := 0; i < length; i += 2 {
		tmp0 := (in[i<<1] + in[i<<1+1]) >> 15
		tmp1 := (in[(i<<1)+2] + in[(i<<1)+3]) >> 15
		if tmp0 > 0x00007FFF {
			tmp0 = 0x00007FFF
		}
		if tmp0 < -0x00008000 {
			tmp0 = -0x00008000
		}
		out[i] = int16(tmp0)
		if tmp1 > 0x00007FFF {
			tmp1 = 0x00007FFF
		}
		if tmp1 < -0x00008000 {
			tmp1 = -0x00008000
		}
		out[i+1] = int16(tmp1)
	}
}

// downBy2ShortToInt decimates int16 input to int32 output by factor 2.
// Input:  int16
// Output: int32 (shifted 15 positions to the left, + offset 16384) (of length len/2)
// State:  filter state array; length = 8
func downBy2ShortToInt(in []int16, out []int32, state []int32) {
	length := len(in) >> 1

	// lower allpass filter (operates on even input samples)
	for i := 0; i < length; i++ {
		tmp0 := (int32(in[i<<1]) << 15) + (1 << 14)
		// overflow: matches C RTC_NO_SANITIZE("signed-integer-overflow")
		diff := tmp0 - state[1]
		diff = (diff + (1 << 13)) >> 14
		tmp1 := state[0] + diff*int32(kResampleAllpass[1][0])
		state[0] = tmp0
		diff = tmp1 - state[2]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		tmp0 = state[1] + diff*int32(kResampleAllpass[1][1])
		state[1] = tmp1
		diff = tmp0 - state[3]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		state[3] = state[2] + diff*int32(kResampleAllpass[1][2])
		state[2] = tmp0

		out[i] = state[3] >> 1
	}

	// upper allpass filter (operates on odd input samples)
	for i := 0; i < length; i++ {
		tmp0 := (int32(in[i<<1+1]) << 15) + (1 << 14)
		// overflow: matches C RTC_NO_SANITIZE("signed-integer-overflow")
		diff := tmp0 - state[5]
		diff = (diff + (1 << 13)) >> 14
		tmp1 := state[4] + diff*int32(kResampleAllpass[0][0])
		state[4] = tmp0
		diff = tmp1 - state[6]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		tmp0 = state[5] + diff*int32(kResampleAllpass[0][1])
		state[5] = tmp1
		diff = tmp0 - state[7]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		state[7] = state[6] + diff*int32(kResampleAllpass[0][2])
		state[6] = tmp0

		out[i] += state[7] >> 1
	}
}

// lpBy2IntToInt performs lowpass filtering and decimation by 2.
// Input:  int32 (shifted 15 positions to the left, + offset 16384)
// Output: int32 (normalized, not saturated)
// State:  filter state array; length = 16
func lpBy2IntToInt(in []int32, out []int32, state []int32) {
	length := len(in) >> 1

	// lower allpass filter: odd input -> even output samples
	tmp0 := state[12]
	for i := 0; i < length; i++ {
		// overflow: matches C RTC_NO_SANITIZE("signed-integer-overflow")
		diff := tmp0 - state[1]
		diff = (diff + (1 << 13)) >> 14
		tmp1 := state[0] + diff*int32(kResampleAllpass[1][0])
		state[0] = tmp0
		diff = tmp1 - state[2]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		tmp0 = state[1] + diff*int32(kResampleAllpass[1][1])
		state[1] = tmp1
		diff = tmp0 - state[3]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		state[3] = state[2] + diff*int32(kResampleAllpass[1][2])
		state[2] = tmp0

		out[i<<1] = state[3] >> 1
		tmp0 = in[i<<1+1] // odd input
	}

	// upper allpass filter: even input -> even output samples
	for i := 0; i < length; i++ {
		tmp0 = in[i<<1]
		// overflow: matches C RTC_NO_SANITIZE("signed-integer-overflow")
		diff := tmp0 - state[5]
		diff = (diff + (1 << 13)) >> 14
		tmp1 := state[4] + diff*int32(kResampleAllpass[0][0])
		state[4] = tmp0
		diff = tmp1 - state[6]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		tmp0 = state[5] + diff*int32(kResampleAllpass[0][1])
		state[5] = tmp1
		diff = tmp0 - state[7]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		state[7] = state[6] + diff*int32(kResampleAllpass[0][2])
		state[6] = tmp0

		out[i<<1] = (out[i<<1] + (state[7] >> 1)) >> 15
	}

	// lower allpass filter: even input -> odd output samples
	for i := 0; i < length; i++ {
		tmp0 = in[i<<1]
		diff := tmp0 - state[9]
		diff = (diff + (1 << 13)) >> 14
		tmp1 := state[8] + diff*int32(kResampleAllpass[1][0])
		state[8] = tmp0
		diff = tmp1 - state[10]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		tmp0 = state[9] + diff*int32(kResampleAllpass[1][1])
		state[9] = tmp1
		diff = tmp0 - state[11]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		state[11] = state[10] + diff*int32(kResampleAllpass[1][2])
		state[10] = tmp0

		out[i<<1+1] = state[11] >> 1
	}

	// upper allpass filter: odd input -> odd output samples
	for i := 0; i < length; i++ {
		tmp0 = in[i<<1+1]
		diff := tmp0 - state[13]
		diff = (diff + (1 << 13)) >> 14
		tmp1 := state[12] + diff*int32(kResampleAllpass[0][0])
		state[12] = tmp0
		diff = tmp1 - state[14]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		tmp0 = state[13] + diff*int32(kResampleAllpass[0][1])
		state[13] = tmp1
		diff = tmp0 - state[15]
		diff = diff >> 14
		if diff < 0 {
			diff++
		}
		state[15] = state[14] + diff*int32(kResampleAllpass[0][2])
		state[14] = tmp0

		out[i<<1+1] = (out[i<<1+1] + (state[15] >> 1)) >> 15
	}
}
