// Copyright (c) 2012 The WebRTC project authors. All Rights Reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package vad

// Feature calculating functionality used in vad_core.

// 160*log10(2) in Q9.
const kLogConst = 24660

// 14 in Q10.
const kLogEnergyIntPart = 14336

// High pass filter coefficients in Q14.
// Read-only at runtime.
var kHpZeroCoefs = [3]int16{6631, -13262, 6631}
var kHpPoleCoefs = [3]int16{16384, -7756, 5620}

// Allpass filter coefficients for split filter, upper and lower, in Q15.
// Upper: 0.64, Lower: 0.17.
var kAllPassCoefsQ15 = [2]int16{20972, 5571}

// Adjustment for division with two in SplitFilter.
var kOffsetVector = [6]int16{368, 368, 272, 176, 176, 176}

// highPassFilter performs high pass filtering with a cut-off frequency at 80 Hz,
// assuming data_in is sampled at 500 Hz.
func highPassFilter(dataIn []int16, filterState []int16, dataOut []int16) {
	for i := range dataIn {
		// All-zero section (filter coefficients in Q14).
		tmp32 := int32(kHpZeroCoefs[0]) * int32(dataIn[i])
		tmp32 += int32(kHpZeroCoefs[1]) * int32(filterState[0])
		tmp32 += int32(kHpZeroCoefs[2]) * int32(filterState[1])
		filterState[1] = filterState[0]
		filterState[0] = dataIn[i]

		// All-pole section (filter coefficients in Q14).
		tmp32 -= int32(kHpPoleCoefs[1]) * int32(filterState[2])
		tmp32 -= int32(kHpPoleCoefs[2]) * int32(filterState[3])
		filterState[3] = filterState[2]
		filterState[2] = int16(tmp32 >> 14)
		dataOut[i] = filterState[2]
	}
}

// allPassFilter performs all pass filtering with a given coefficient in Q15.
// dataIn and dataOut must NOT overlap.
// Input: Q0, Output: Q(-1).
func allPassFilter(dataIn []int16, filterCoefficient int16, filterState *int16, dataOut []int16) {
	// State is in Q15 (stored as Q(-1) scaled up by 2^16).
	state32 := int32(*filterState) << 16

	for i := 0; i < len(dataIn); i++ {
		tmp32 := state32 + int32(filterCoefficient)*int32(dataIn[i])
		tmp16 := int16(tmp32 >> 16) // Q(-1)
		dataOut[i] = tmp16
		state32 = int32(dataIn[i])<<14 - int32(filterCoefficient)*int32(tmp16) // Q14
		state32 *= 2                                                           // Q15
	}

	*filterState = int16(state32 >> 16) // Q(-1)
}

// splitFilter splits dataIn into hpDataOut (high pass) and lpDataOut (low pass)
// corresponding to upper and lower halves of the spectrum.
// dataIn length must be even.
func splitFilter(dataIn []int16, upperState *int16, lowerState *int16,
	hpDataOut []int16, lpDataOut []int16) {
	halfLength := len(dataIn) >> 1

	// All-pass filtering upper branch (even-indexed samples: 0, 2, 4, ...).
	allPassFilterStride(dataIn, halfLength, kAllPassCoefsQ15[0], upperState, hpDataOut, 0)

	// All-pass filtering lower branch (odd-indexed samples: 1, 3, 5, ...).
	allPassFilterStride(dataIn, halfLength, kAllPassCoefsQ15[1], lowerState, lpDataOut, 1)

	// Make LP and HP signals.
	for i := 0; i < halfLength; i++ {
		tmpOut := hpDataOut[i]
		hpDataOut[i] -= lpDataOut[i]
		lpDataOut[i] += tmpOut
	}
}

// allPassFilterStride performs all-pass filtering on strided samples.
// stride=0 for even-indexed (0,2,4,...), stride=1 for odd-indexed (1,3,5,...).
func allPassFilterStride(dataIn []int16, length int, filterCoefficient int16, filterState *int16, dataOut []int16, stride int) {
	state32 := int32(*filterState) << 16
	for i := 0; i < length; i++ {
		idx := i*2 + stride
		tmp32 := state32 + int32(filterCoefficient)*int32(dataIn[idx])
		tmp16 := int16(tmp32 >> 16)
		dataOut[i] = tmp16
		state32 = int32(dataIn[idx])<<14 - int32(filterCoefficient)*int32(tmp16)
		state32 *= 2
	}
	*filterState = int16(state32 >> 16)
}

// logOfEnergy calculates the energy of dataIn in dB (Q4), and also updates
// totalEnergy if necessary.
func logOfEnergy(dataIn []int16, offset int16, totalEnergy *int16, logEnergy *int16) {
	totRshifts := 0

	// Compute energy with scaling.
	energyVal := uint32(energy(dataIn, &totRshifts))

	if energyVal != 0 {
		// Normalize to 15 bits.
		normalizingRshifts := 17 - int(normU32(energyVal))
		log2Energy := int16(kLogEnergyIntPart)

		totRshifts += normalizingRshifts
		if normalizingRshifts < 0 {
			energyVal <<= -normalizingRshifts
		} else {
			energyVal >>= normalizingRshifts
		}

		// Add fractional part to log2Energy.
		log2Energy += int16((energyVal & 0x00003FFF) >> 4)

		// kLogConst is in Q9, log2Energy in Q10, totRshifts in Q0.
		// Output in Q4.
		*logEnergy = int16(((int32(kLogConst) * int32(log2Energy)) >> 19) +
			((int32(totRshifts) * int32(kLogConst)) >> 9))

		if *logEnergy < 0 {
			*logEnergy = 0
		}
	} else {
		*logEnergy = offset
		return
	}

	*logEnergy += offset

	// Update approximate totalEnergy.
	if *totalEnergy <= kMinEnergy {
		if totRshifts >= 0 {
			*totalEnergy += kMinEnergy + 1
		} else {
			*totalEnergy += int16(energyVal >> -totRshifts)
		}
	}
}

// calculateFeatures takes dataLength samples of dataIn and calculates the
// logarithm of the energy of each of the 6 frequency bands used by the VAD.
// Features are given in Q4. Returns approximate total energy.
func calculateFeatures(self *vadInstT, dataIn []int16, dataLength int, features []int16) int16 {
	var totalEnergy int16

	halfDataLength := dataLength >> 1
	length := halfDataLength

	// Temporary buffers.
	var hp120 [120]int16
	var lp120 [120]int16
	var hp60 [60]int16
	var lp60 [60]int16

	// Split at 2000 Hz and downsample.
	frequencyBand := 0
	inPtr := dataIn[:dataLength]
	splitFilter(inPtr, &self.upperState[frequencyBand], &self.lowerState[frequencyBand],
		hp120[:length], lp120[:length])

	// Upper band [2000-4000]: split at 3000 Hz and downsample.
	frequencyBand = 1
	inPtr = hp120[:halfDataLength]
	splitFilter(inPtr, &self.upperState[frequencyBand], &self.lowerState[frequencyBand],
		hp60[:length>>1], lp60[:length>>1])

	// Energy in 3000-4000 Hz.
	length >>= 1

	logOfEnergy(hp60[:length], kOffsetVector[5], &totalEnergy, &features[5])
	// Energy in 2000-3000 Hz.
	logOfEnergy(lp60[:length], kOffsetVector[4], &totalEnergy, &features[4])

	// Lower band [0-2000]: split at 1000 Hz and downsample.
	frequencyBand = 2
	inPtr = lp120[:halfDataLength]
	length = halfDataLength
	splitFilter(inPtr, &self.upperState[frequencyBand], &self.lowerState[frequencyBand],
		hp60[:length>>1], lp60[:length>>1])

	// Energy in 1000-2000 Hz.
	length >>= 1
	logOfEnergy(hp60[:length], kOffsetVector[3], &totalEnergy, &features[3])

	// Lower band [0-1000]: split at 500 Hz and downsample.
	frequencyBand = 3
	inPtr = lp60[:length]
	hpOut := hp120[:length>>1]
	lpOut := lp120[:length>>1]
	splitFilter(inPtr, &self.upperState[frequencyBand], &self.lowerState[frequencyBand],
		hpOut, lpOut)

	// Energy in 500-1000 Hz.
	length >>= 1
	logOfEnergy(hp120[:length], kOffsetVector[2], &totalEnergy, &features[2])

	// Lower band [0-500]: split at 250 Hz and downsample.
	frequencyBand = 4
	inPtr = lp120[:length]
	hpOut2 := hp60[:length>>1]
	lpOut2 := lp60[:length>>1]
	splitFilter(inPtr, &self.upperState[frequencyBand], &self.lowerState[frequencyBand],
		hpOut2, lpOut2)

	// Energy in 250-500 Hz.
	length >>= 1
	logOfEnergy(hp60[:length], kOffsetVector[1], &totalEnergy, &features[1])

	// Remove 0-80 Hz by high pass filtering the lower band.
	highPassFilter(lp60[:length], self.hpFilterState[:], hp120[:length])

	// Energy in 80-250 Hz.
	logOfEnergy(hp120[:length], kOffsetVector[0], &totalEnergy, &features[0])

	return totalEnergy
}
