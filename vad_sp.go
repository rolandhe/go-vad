// Copyright (c) 2012 The WebRTC project authors. All Rights Reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package vad

// Signal processing tools used in vad_core.

// Allpass filter coefficients for downsampling, upper and lower, in Q13.
// Upper: 0.64, Lower: 0.17.
// Read-only at runtime.
var kAllPassCoefsQ13 = [2]int16{5243, 1392}

// Smoothing constants in Q15.
// 0.2 and 0.99 respectively.
const (
	kSmoothingDown = 6553
	kSmoothingUp   = 32439
)

// downsampling downsamples the signal by a factor 2 (e.g. 32->16 or 16->8).
// Filter coefficients in Q13, filter state in Q0.
func downsampling(signalIn []int16, signalOut []int16, filterState []int32) {
	halfLength := len(signalIn) >> 1
	tmp32_1 := filterState[0]
	tmp32_2 := filterState[1]
	inIdx := 0

	for n := 0; n < halfLength; n++ {
		// All-pass filtering upper branch.
		tmp16_1 := int16((tmp32_1 >> 1) +
			((int32(kAllPassCoefsQ13[0]) * int32(signalIn[inIdx])) >> 14))
		signalOut[n] = tmp16_1
		tmp32_1 = int32(signalIn[inIdx]) - ((int32(kAllPassCoefsQ13[0]) * int32(tmp16_1)) >> 12)
		inIdx++

		// All-pass filtering lower branch.
		tmp16_2 := int16((tmp32_2 >> 1) +
			((int32(kAllPassCoefsQ13[1]) * int32(signalIn[inIdx])) >> 14))
		signalOut[n] += tmp16_2
		tmp32_2 = int32(signalIn[inIdx]) - ((int32(kAllPassCoefsQ13[1]) * int32(tmp16_2)) >> 12)
		inIdx++
	}

	filterState[0] = tmp32_1
	filterState[1] = tmp32_2
}

// findMinimum maintains the 16 smallest feature values over a 100-frame window
// and returns the smoothed median of the 5 smallest values.
func findMinimum(self *vadInstT, featureValue int16, channel int) int16 {
	offset := channel << 4 // 16 values per channel
	age := self.indexVector[offset : offset+16]
	smallestValues := self.lowValueVector[offset : offset+16]

	// Each value in smallestValues gets 1 loop older.
	for i := 0; i < 16; i++ {
		if age[i] != 100 {
			age[i]++
		} else {
			// Too old, remove and shift larger values downward.
			for j := i; j < 15; j++ {
				smallestValues[j] = smallestValues[j+1]
				age[j] = age[j+1]
			}
			age[15] = 101
			smallestValues[15] = 10000
		}
	}

	// Binary search to find insertion position for featureValue.
	position := -1
	if featureValue < smallestValues[7] {
		if featureValue < smallestValues[3] {
			if featureValue < smallestValues[1] {
				if featureValue < smallestValues[0] {
					position = 0
				} else {
					position = 1
				}
			} else if featureValue < smallestValues[2] {
				position = 2
			} else {
				position = 3
			}
		} else if featureValue < smallestValues[5] {
			if featureValue < smallestValues[4] {
				position = 4
			} else {
				position = 5
			}
		} else if featureValue < smallestValues[6] {
			position = 6
		} else {
			position = 7
		}
	} else if featureValue < smallestValues[15] {
		if featureValue < smallestValues[11] {
			if featureValue < smallestValues[9] {
				if featureValue < smallestValues[8] {
					position = 8
				} else {
					position = 9
				}
			} else if featureValue < smallestValues[10] {
				position = 10
			} else {
				position = 11
			}
		} else if featureValue < smallestValues[13] {
			if featureValue < smallestValues[12] {
				position = 12
			} else {
				position = 13
			}
		} else if featureValue < smallestValues[14] {
			position = 14
		} else {
			position = 15
		}
	}

	// Insert new small value at correct position.
	if position > -1 {
		for i := 15; i > position; i-- {
			smallestValues[i] = smallestValues[i-1]
			age[i] = age[i-1]
		}
		smallestValues[position] = featureValue
		age[position] = 1
	}

	// Get current median.
	var currentMedian int16
	if self.frameCounter > 2 {
		currentMedian = smallestValues[2]
	} else if self.frameCounter > 0 {
		currentMedian = smallestValues[0]
	} else {
		currentMedian = 1600
	}

	// Smooth the median value.
	var alpha int16
	if self.frameCounter > 0 {
		if currentMedian < self.meanValue[channel] {
			alpha = kSmoothingDown // 0.2 in Q15
		} else {
			alpha = kSmoothingUp // 0.99 in Q15
		}
	}
	tmp32 := (int32(alpha) + 1) * int32(self.meanValue[channel])
	tmp32 += int32(word16Max-alpha) * int32(currentMedian)
	tmp32 += 16384
	self.meanValue[channel] = int16(tmp32 >> 15)

	return self.meanValue[channel]
}
