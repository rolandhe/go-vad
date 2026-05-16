// Copyright (c) 2012 The WebRTC project authors. All Rights Reserved.
// Copyright (c) 2016 Daniel Pirch.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

// Package vad implements Voice Activity Detection for 16-bit PCM audio.
//
// It is a pure Go port of libfvad (https://github.com/dpirch/libfvad),
// which is itself a fork of the WebRTC VAD engine.
//
// The VAD processes audio frames of 10, 20, or 30 ms duration at
// sample rates of 8000, 16000, 32000, or 48000 Hz.
package vad

import "fmt"

// Mode defines the VAD aggressiveness level.
// Higher modes are more restrictive in reporting speech.
type Mode int

const (
	ModeQuality        Mode = 0 // Normal quality (default)
	ModeLowBitrate     Mode = 1 // Low bitrate
	ModeAggressive     Mode = 2 // Aggressive
	ModeVeryAggressive Mode = 3 // Very aggressive
)

// SampleRate defines a valid input sample rate in Hz.
type SampleRate int

const (
	SampleRate8k  SampleRate = 8000
	SampleRate16k SampleRate = 16000
	SampleRate32k SampleRate = 32000
	SampleRate48k SampleRate = 48000
)

// Result is a VAD decision for a single audio frame.
type Result int

const (
	ResultNoVoice Result = 0 // No active voice detected
	ResultVoice   Result = 1 // Active voice detected
)

// Valid sample rates in kHz.
var validRates = []int{8, 16, 32, 48}

// VAD process functions for each valid sample rate.
var processFuncs = []func(*vadInstT, []int16, int) int{
	calcVad8khz,
	calcVad16khz,
	calcVad32khz,
	calcVad48khz,
}

// Valid frame lengths in ms.
var validFrameTimes = []int{10, 20, 30}

// VAD is a Voice Activity Detection instance.
// A VAD instance should be created with New and freed when no longer needed.
type VAD struct {
	core    vadInstT
	rateIdx int // index in validRates
}

// New creates and initializes a new VAD instance.
// The default mode is ModeQuality and the default sample rate is 8000 Hz.
func New() *VAD {
	v := &VAD{}
	v.Reset()
	return v
}

// Reset reinitializes the VAD instance, clearing all state and resetting
// mode and sample rate to defaults.
func (v *VAD) Reset() {
	initCore(&v.core)
	v.rateIdx = 0
}

// SetMode sets the VAD operating mode (aggressiveness).
// Valid modes are 0 (Quality), 1 (Low bitrate), 2 (Aggressive),
// and 3 (Very aggressive).
func (v *VAD) SetMode(mode Mode) error {
	if mode < 0 || mode > 3 {
		return fmt.Errorf("vad: invalid mode %d (valid: 0-3)", mode)
	}
	if setModeCore(&v.core, int(mode)) != 0 {
		return fmt.Errorf("vad: failed to set mode %d", mode)
	}
	return nil
}

// SetSampleRate sets the input sample rate in Hz.
// Valid values are 8000, 16000, 32000, and 48000.
func (v *VAD) SetSampleRate(sr SampleRate) error {
	for i, rate := range validRates {
		if rate*1000 == int(sr) {
			v.rateIdx = i
			return nil
		}
	}
	return fmt.Errorf("vad: invalid sample rate %d (valid: 8000, 16000, 32000, 48000)", sr)
}

// validLength checks if the frame length is valid for the current sample rate.
func (v *VAD) validLength(length int) bool {
	samplesPerMs := validRates[v.rateIdx]
	for _, ft := range validFrameTimes {
		if ft*samplesPerMs == length {
			return true
		}
	}
	return false
}

// Process processes a frame of audio samples and returns the VAD decision.
//
// The frame must contain signed 16-bit PCM samples. The frame length must
// correspond to 10, 20, or 30 ms at the configured sample rate. For example,
// at 8 kHz, valid lengths are 80, 160, and 240 samples.
//
// Returns ResultVoice (1) if active voice is detected, ResultNoVoice (0) if
// no voice is detected, or an error if the frame length is invalid.
func (v *VAD) Process(frame []int16) (Result, error) {
	length := len(frame)
	if !v.validLength(length) {
		return 0, fmt.Errorf("vad: invalid frame length %d for sample rate %d Hz (valid: 10, 20, or 30 ms)",
			length, validRates[v.rateIdx]*1000)
	}

	result := processFuncs[v.rateIdx](&v.core, frame, length)
	if result > 0 {
		result = 1
	}
	return Result(result), nil
}
