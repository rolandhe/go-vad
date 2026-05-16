// Copyright (c) 2011 The WebRTC project authors. All Rights Reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package vad

// webRtcSplState48khzTo8khz holds state for the 48 kHz -> 8 kHz resampler.
type webRtcSplState48khzTo8khz struct {
	S_48_24 [8]int32
	S_24_24 [16]int32
	S_24_16 [8]int32
	S_16_8  [8]int32
}

// resample48khzTo8khz resamples 48 kHz input to 8 kHz output.
// Processes one 10ms frame: 480 input samples -> 80 output samples.
// tmpmem must be at least 480+256 = 736 int32 elements.
// in must have at least 480 samples (caller guarantees this).
func resample48khzTo8khz(in []int16, out []int16, state *webRtcSplState48khzTo8khz, tmpmem []int32) {
	// 48 --> 24 (int16 to int32, decimate by 2)
	downBy2ShortToInt(in[:480], tmpmem[256:256+240], state.S_48_24[:])

	// 24 --> 24(LP) (int32 to int32, lowpass + decimate by 2)
	lpBy2IntToInt(tmpmem[256:256+240], tmpmem[16:16+240], state.S_24_24[:])

	// 24 --> 16 (fractional resample 48->32, ratio 2/3)
	copy(tmpmem[8:16], state.S_24_16[:])
	copy(state.S_24_16[:], tmpmem[248:256])
	// Input needs 3*K+7 = 247 elements, output needs 2*K = 160.
	resample48khzTo32khz(tmpmem[8:255], tmpmem[0:160], 80)

	// 16 --> 8 (int32 to int16, decimate by 2)
	downBy2IntToShort(tmpmem[0:160], out, state.S_16_8[:])
}

// resetResample48khzTo8khz zeroes all state for the 48->8 resampler.
func resetResample48khzTo8khz(state *webRtcSplState48khzTo8khz) {
	for i := range state.S_48_24 {
		state.S_48_24[i] = 0
	}
	for i := range state.S_24_24 {
		state.S_24_24[i] = 0
	}
	for i := range state.S_24_16 {
		state.S_24_16[i] = 0
	}
	for i := range state.S_16_8 {
		state.S_16_8[i] = 0
	}
}
