// Copyright (c) 2012 The WebRTC project authors. All Rights Reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package vad

// VAD constants.
const (
	kNumChannels  = 6 // Number of frequency bands (named channels).
	kNumGaussians = 2 // Number of Gaussians per channel in the GMM.
	kTableSize    = kNumChannels * kNumGaussians
	kMinEnergy    = 10 // Minimum energy required to trigger audio signal.
)

// vadInstT holds all state for the VAD core.
type vadInstT struct {
	vad                      int
	downsamplingFilterStates [4]int32
	state48To8               webRtcSplState48khzTo8khz
	noiseMeans               [kTableSize]int16
	speechMeans              [kTableSize]int16
	noiseStds                [kTableSize]int16
	speechStds               [kTableSize]int16
	frameCounter             int32
	overHang                 int16
	numOfSpeech              int16
	indexVector              [16 * kNumChannels]int16
	lowValueVector           [16 * kNumChannels]int16
	meanValue                [kNumChannels]int16
	upperState               [5]int16
	lowerState               [5]int16
	hpFilterState            [4]int16
	overHangMax1             [3]int16
	overHangMax2             [3]int16
	individual               [3]int16
	total                    [3]int16
	initFlag                 int
}
