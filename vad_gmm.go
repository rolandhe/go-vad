// Copyright (c) 2011 The WebRTC project authors. All Rights Reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package vad

// Gaussian probability calculations internally used in vad_core.

const (
	kCompVar = 22005
	// log2(exp(1)) in Q12
	kLog2Exp = 5909
)

// gaussianProbability calculates the probability for |input|, given that
// |input| comes from a normal distribution with mean and std.
//
// Inputs:
//   - input : input sample in Q4
//   - mean  : mean input in the statistical model, Q7
//   - std   : standard deviation, Q7
//
// Output:
//   - delta : input used when updating the model, Q11.
//     delta = (input - mean) / std^2
//
// Return: probability for |input| in Q20.
//
//	1 / std * exp(-(input - mean)^2 / (2 * std^2))
func gaussianProbability(input int16, mean int16, std int16, delta *int16) int32 {
	var invStd, invStd2, expValue int16

	// Calculate invStd = 1 / s, in Q10.
	// 131072 = 1 in Q17, (std >> 1) is for rounding.
	// Q-domain: Q17 / Q7 = Q10.
	tmp32 := int32(131072) + int32(std>>1)
	invStd = int16(divW32W16(tmp32, std))

	// Calculate invStd2 = 1 / s^2, in Q14.
	tmp16 := invStd >> 2                                // Q10 -> Q8.
	invStd2 = int16((int32(tmp16) * int32(tmp16)) >> 2) // Q8*Q8 >> 2 = Q14.

	tmp16 = input << 3   // Q4 -> Q7
	tmp16 = tmp16 - mean // Q7 - Q7 = Q7

	// delta = (x - m) / s^2, in Q11.
	// Q14 * Q7 >> 10 = Q11.
	*delta = int16((int32(invStd2) * int32(tmp16)) >> 10)

	// Calculate the exponent tmp32 = (x - m)^2 / (2 * s^2), in Q10.
	// Q11 * Q7 >> 9 = Q10 (note: >> 9 instead of >> 8 accounts for the /2).
	tmp32 = (int32(*delta) * int32(tmp16)) >> 9

	// If exponent is small enough for non-zero probability, calculate
	// expValue ~= exp(-(x - m)^2 / (2 * s^2))
	//          ~= exp2(-log2(exp(1)) * tmp32)
	if tmp32 < kCompVar {
		// Calculate tmp16 = log2(exp(1)) * tmp32, in Q10.
		// Q12 * Q10 >> 12 = Q10.
		tmp16 = int16((kLog2Exp * tmp32) >> 12)
		tmp16 = -tmp16
		expValue = int16(0x0400 | (tmp16 & 0x03FF))
		tmp16 ^= -1 // 0xFFFF in 16-bit two's complement
		tmp16 >>= 10
		tmp16 += 1
		// Get expValue = exp(-tmp32) in Q10.
		expValue >>= tmp16
	}

	// Calculate and return (1 / s) * exp(...), in Q20.
	// Q10 * Q10 = Q20.
	return int32(invStd) * int32(expValue)
}
