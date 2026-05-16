// Copyright (c) 2011 The WebRTC project authors. All Rights Reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package vad

// Interpolation coefficients for 48kHz -> 32kHz resampling (ratio 2/3).
// Read-only at runtime.
var kCoefficients48To32 = [2][8]int16{
	{778, -2050, 1087, 23285, 12903, -3783, 441, 222},
	{222, 441, -3783, 12903, 23285, 1087, -2050, 778},
}

// resample48khzTo32khz resamples 48 kHz -> 32 kHz.
// Input:  int32 (normalized, not saturated) :: size 3*K
// Output: int32 (shifted 15 positions to the left, + offset 16384) :: size 2*K
func resample48khzTo32khz(in []int32, out []int32, K int) {
	inIdx := 0
	outIdx := 0
	for m := 0; m < K; m++ {
		tmp := int32(1 << 14)
		tmp += int32(kCoefficients48To32[0][0]) * in[inIdx+0]
		tmp += int32(kCoefficients48To32[0][1]) * in[inIdx+1]
		tmp += int32(kCoefficients48To32[0][2]) * in[inIdx+2]
		tmp += int32(kCoefficients48To32[0][3]) * in[inIdx+3]
		tmp += int32(kCoefficients48To32[0][4]) * in[inIdx+4]
		tmp += int32(kCoefficients48To32[0][5]) * in[inIdx+5]
		tmp += int32(kCoefficients48To32[0][6]) * in[inIdx+6]
		tmp += int32(kCoefficients48To32[0][7]) * in[inIdx+7]
		out[outIdx+0] = tmp

		tmp = int32(1 << 14)
		tmp += int32(kCoefficients48To32[1][0]) * in[inIdx+1]
		tmp += int32(kCoefficients48To32[1][1]) * in[inIdx+2]
		tmp += int32(kCoefficients48To32[1][2]) * in[inIdx+3]
		tmp += int32(kCoefficients48To32[1][3]) * in[inIdx+4]
		tmp += int32(kCoefficients48To32[1][4]) * in[inIdx+5]
		tmp += int32(kCoefficients48To32[1][5]) * in[inIdx+6]
		tmp += int32(kCoefficients48To32[1][6]) * in[inIdx+7]
		tmp += int32(kCoefficients48To32[1][7]) * in[inIdx+8]
		out[outIdx+1] = tmp

		inIdx += 3
		outIdx += 2
	}
}
