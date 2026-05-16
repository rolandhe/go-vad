// Copyright (c) 2012 The WebRTC project authors. All Rights Reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package vad

// Spectrum Weighting.
// Read-only at runtime — copied to instance fields by initCore.
var kSpectrumWeight = [kNumChannels]int16{6, 8, 10, 12, 14, 16}

const (
	kNoiseUpdateConst  = 655  // Q15
	kSpeechUpdateConst = 6554 // Q15
	kBackEta           = 154  // Q8
)

// Minimum difference between the two models, Q5.
var kMinimumDifference = [kNumChannels]int16{544, 544, 576, 576, 576, 576}

// Upper limit of mean value for speech model, Q7.
var kMaximumSpeech = [kNumChannels]int16{11392, 11392, 11520, 11520, 11520, 11520}

// Minimum value for mean value.
var kMinimumMean = [kNumGaussians]int16{640, 768}

// Upper limit of mean value for noise model, Q7.
var kMaximumNoise = [kNumChannels]int16{9216, 9088, 8960, 8832, 8704, 8576}

// Start values for the Gaussian models.
// Weights for the two Gaussians for the six channels (noise), Q7.
var kNoiseDataWeights = [kTableSize]int16{34, 62, 72, 66, 53, 25, 94, 66, 56, 62, 75, 103}

// Weights for the two Gaussians for the six channels (speech), Q7.
var kSpeechDataWeights = [kTableSize]int16{48, 82, 45, 87, 50, 47, 80, 46, 83, 41, 78, 81}

// Means for the two Gaussians for the six channels (noise), Q7.
var kNoiseDataMeans = [kTableSize]int16{6738, 4892, 7065, 6715, 6771, 3369, 7646, 3863, 7820, 7266, 5020, 4362}

// Means for the two Gaussians for the six channels (speech), Q7.
var kSpeechDataMeans = [kTableSize]int16{8306, 10085, 10078, 11823, 11843, 6309, 9473, 9571, 10879, 7581, 8180, 7483}

// Stds for the two Gaussians for the six channels (noise), Q7.
var kNoiseDataStds = [kTableSize]int16{378, 1064, 493, 582, 688, 593, 474, 697, 475, 688, 421, 455}

// Stds for the two Gaussians for the six channels (speech), Q7.
var kSpeechDataStds = [kTableSize]int16{555, 505, 567, 524, 585, 1231, 509, 828, 492, 1540, 1079, 850}

// Maximum number of counted speech (VAD = 1) frames in a row.
const kMaxSpeechFrames = 6

// Minimum standard deviation for both speech and noise.
const kMinStd = 384

// Default aggressiveness mode.
const kDefaultMode = 0
const kInitCheck = 42

// Mode 0, Quality.
var kOverHangMax1Q = [3]int16{8, 4, 3}
var kOverHangMax2Q = [3]int16{14, 7, 5}
var kLocalThresholdQ = [3]int16{24, 21, 24}
var kGlobalThresholdQ = [3]int16{57, 48, 57}

// Mode 1, Low bitrate.
var kOverHangMax1LBR = [3]int16{8, 4, 3}
var kOverHangMax2LBR = [3]int16{14, 7, 5}
var kLocalThresholdLBR = [3]int16{37, 32, 37}
var kGlobalThresholdLBR = [3]int16{100, 80, 100}

// Mode 2, Aggressive.
var kOverHangMax1AGG = [3]int16{6, 3, 2}
var kOverHangMax2AGG = [3]int16{9, 5, 3}
var kLocalThresholdAGG = [3]int16{82, 78, 82}
var kGlobalThresholdAGG = [3]int16{285, 260, 285}

// Mode 3, Very aggressive.
var kOverHangMax1VAG = [3]int16{6, 3, 2}
var kOverHangMax2VAG = [3]int16{9, 5, 3}
var kLocalThresholdVAG = [3]int16{94, 94, 94}
var kGlobalThresholdVAG = [3]int16{1100, 1050, 1100}

// weightedAverage calculates the weighted average w.r.t. number of Gaussians.
// data is the full array (e.g., noiseMeans). channel is the frequency band index.
// data and weights are modified — data entries are updated with offset.
func weightedAverage(data []int16, channel int, offset int16, weights []int16) int32 {
	var weightedAve int32
	for k := 0; k < kNumGaussians; k++ {
		idx := channel + k*kNumChannels
		data[idx] += offset
		weightedAve += int32(data[idx]) * int32(weights[idx])
	}
	return weightedAve
}

// overflowingMulS16ByS32ToS32 multiplies int16 by int32 with intentional overflow.
func overflowingMulS16ByS32ToS32(a int16, b int32) int32 {
	return int32(a) * b
}

// gmmProbability calculates the probabilities for both speech and background noise
// using Gaussian Mixture Models, performs a hypothesis-test, and returns the VAD decision.
func gmmProbability(self *vadInstT, features []int16, totalPower int16, frameLength int) int16 {
	var h0, h1 int16
	var logLikelihoodRatio int16
	var vadflag int16
	var shiftsH0, shiftsH1 int16
	var tmpS16, tmp1S16, tmp2S16 int16
	var diff int16
	var nmk, nmk2, nmk3, smk, smk2, nsk, ssk int16
	var delt, ndelt int16
	var maxspe, maxmu int16
	var deltaN [kTableSize]int16
	var deltaS [kTableSize]int16
	var ngprvec [kTableSize]int16 // Conditional probability = 0.
	var sgprvec [kTableSize]int16 // Conditional probability = 0.
	var h0Test, h1Test int32
	var tmp1S32, tmp2S32 int32
	var sumLogLikelihoodRatios int32
	var noiseGlobalMean, speechGlobalMean int32
	var noiseProbability [kNumGaussians]int32
	var speechProbability [kNumGaussians]int32
	var overhead1, overhead2, individualTest, totalTest int16

	// Set various thresholds based on frame lengths (80, 160 or 240 samples).
	switch frameLength {
	case 80:
		overhead1 = self.overHangMax1[0]
		overhead2 = self.overHangMax2[0]
		individualTest = self.individual[0]
		totalTest = self.total[0]
	case 160:
		overhead1 = self.overHangMax1[1]
		overhead2 = self.overHangMax2[1]
		individualTest = self.individual[1]
		totalTest = self.total[1]
	default:
		overhead1 = self.overHangMax1[2]
		overhead2 = self.overHangMax2[2]
		individualTest = self.individual[2]
		totalTest = self.total[2]
	}

	if totalPower > kMinEnergy {
		for channel := 0; channel < kNumChannels; channel++ {
			h0Test = 0
			h1Test = 0
			for k := 0; k < kNumGaussians; k++ {
				gaussian := channel + k*kNumChannels

				// Probability under H0 (noise). Value given in Q27 = Q7 * Q20.
				tmp1S32 = gaussianProbability(features[channel],
					self.noiseMeans[gaussian],
					self.noiseStds[gaussian],
					&deltaN[gaussian])
				noiseProbability[k] = int32(kNoiseDataWeights[gaussian]) * tmp1S32
				h0Test += noiseProbability[k] // Q27

				// Probability under H1 (speech). Value given in Q27 = Q7 * Q20.
				tmp1S32 = gaussianProbability(features[channel],
					self.speechMeans[gaussian],
					self.speechStds[gaussian],
					&deltaS[gaussian])
				speechProbability[k] = int32(kSpeechDataWeights[gaussian]) * tmp1S32
				h1Test += speechProbability[k] // Q27
			}

			// Calculate log likelihood ratio: approx shiftsH0 - shiftsH1.
			shiftsH0 = normW32(h0Test)
			shiftsH1 = normW32(h1Test)
			if h0Test == 0 {
				shiftsH0 = 31
			}
			if h1Test == 0 {
				shiftsH1 = 31
			}
			logLikelihoodRatio = shiftsH0 - shiftsH1

			// Update sum_log_likelihood_ratios with spectrum weighting.
			sumLogLikelihoodRatios += int32(logLikelihoodRatio) * int32(kSpectrumWeight[channel])

			// Local VAD decision.
			if (logLikelihoodRatio * 4) > individualTest {
				vadflag = 1
			}

			// Calculate local noise probabilities used later when updating the GMM.
			h0 = int16(h0Test >> 12) // Q15
			if h0 > 0 {
				tmp1S32 = (noiseProbability[0] & -4096) << 2     // Q29, 0xFFFFF000 as int32
				ngprvec[channel] = int16(divW32W16(tmp1S32, h0)) // Q14
				ngprvec[channel+kNumChannels] = 16384 - ngprvec[channel]
			} else {
				ngprvec[channel] = 16384
			}

			// Calculate local speech probabilities used later when updating the GMM.
			h1 = int16(h1Test >> 12) // Q15
			if h1 > 0 {
				tmp1S32 = (speechProbability[0] & -4096) << 2    // Q29, 0xFFFFF000 as int32
				sgprvec[channel] = int16(divW32W16(tmp1S32, h1)) // Q14
				sgprvec[channel+kNumChannels] = 16384 - sgprvec[channel]
			}
		}

		// Make a global VAD decision.
		if sumLogLikelihoodRatios >= int32(totalTest) {
			vadflag |= 1
		}

		// Update the model parameters.
		maxspe = 12800
		for channel := 0; channel < kNumChannels; channel++ {
			// Get minimum value in past for long term correction in Q4.
			featureMinimum := findMinimum(self, features[channel], channel)

			// Compute the "global" mean.
			noiseGlobalMean = weightedAverage(self.noiseMeans[:], channel, 0,
				kNoiseDataWeights[:])
			tmp1S16 = int16(noiseGlobalMean >> 6) // Q8

			for k := 0; k < kNumGaussians; k++ {
				gaussian := channel + k*kNumChannels

				nmk = self.noiseMeans[gaussian]
				smk = self.speechMeans[gaussian]
				nsk = self.noiseStds[gaussian]
				ssk = self.speechStds[gaussian]

				// Update noise mean vector if the frame consists of noise only.
				nmk2 = nmk
				if vadflag == 0 {
					// Q14 * Q11 >> 11 = Q14.
					delt = int16((int32(ngprvec[gaussian]) * int32(deltaN[gaussian])) >> 11)
					// Q7 + (Q14 * Q15 >> 22) = Q7.
					nmk2 = nmk + int16((int32(delt)*int32(kNoiseUpdateConst))>>22)
				}

				// Long term correction of the noise mean.
				// Q8 - Q8 = Q8.
				ndelt = (featureMinimum << 4) - tmp1S16
				// Q7 + (Q8 * Q8) >> 9 = Q7.
				nmk3 = nmk2 + int16((int32(ndelt)*int32(kBackEta))>>9)

				// Control that the noise mean does not drift too much.
				tmpS16 = int16((k + 5) << 7)
				if nmk3 < tmpS16 {
					nmk3 = tmpS16
				}
				tmpS16 = int16((72 + k - channel) << 7)
				if nmk3 > tmpS16 {
					nmk3 = tmpS16
				}
				self.noiseMeans[gaussian] = nmk3

				if vadflag != 0 {
					// Update speech mean vector.
					delt = int16((int32(sgprvec[gaussian]) * int32(deltaS[gaussian])) >> 11)
					// Q14 * Q15 >> 21 = Q8.
					tmpS16 = int16((int32(delt) * int32(kSpeechUpdateConst)) >> 21)
					// Q7 + (Q8 >> 1) = Q7. With rounding.
					smk2 = smk + ((tmpS16 + 1) >> 1)

					// Control that the speech mean does not drift too much.
					maxmu = maxspe + 640
					if smk2 < kMinimumMean[k] {
						smk2 = kMinimumMean[k]
					}
					if smk2 > maxmu {
						smk2 = maxmu
					}
					self.speechMeans[gaussian] = smk2 // Q7.

					// Q7 >> 3 = Q4. With rounding.
					tmpS16 = (smk + 4) >> 3
					tmpS16 = features[channel] - tmpS16 // Q4
					// Q11 * Q4 >> 3 = Q12.
					tmp1S32 = (int32(deltaS[gaussian]) * int32(tmpS16)) >> 3
					tmp2S32 = tmp1S32 - 4096
					tmpS16 = sgprvec[gaussian] >> 2
					// Q14 >> 2 * Q12 = Q24.
					tmp1S32 = int32(tmpS16) * tmp2S32
					tmp2S32 = tmp1S32 >> 4 // Q20

					// 0.1 * Q20 / Q7 = Q13.
					if tmp2S32 > 0 {
						tmpS16 = int16(divW32W16(tmp2S32, ssk*10))
					} else {
						tmpS16 = int16(divW32W16(-tmp2S32, ssk*10))
						tmpS16 = -tmpS16
					}
					// Divide by 4 giving an update factor of 0.025.
					// Q13 >> 8 = (Q13 >> 6) / 4 = Q7.
					tmpS16 += 128 // Rounding.
					ssk += (tmpS16 >> 8)
					if ssk < kMinStd {
						ssk = kMinStd
					}
					self.speechStds[gaussian] = ssk
				} else {
					// Update GMM variance vectors (noise).
					// Q4 - (Q7 >> 3) = Q4.
					tmpS16 = features[channel] - (nmk >> 3)
					// Q11 * Q4 >> 3 = Q12.
					tmp1S32 = (int32(deltaN[gaussian]) * int32(tmpS16)) >> 3
					tmp1S32 -= 4096

					// Q14 >> 2 * Q12 = Q24.
					tmpS16 = (ngprvec[gaussian] + 2) >> 2
					// overflow: matches C RTC_NO_SANITIZE("signed-integer-overflow")
					tmp2S32 = overflowingMulS16ByS32ToS32(tmpS16, tmp1S32)
					// Q20 * approx 0.001 (2^-10=0.0009766), hence Q24 >> 14 = Q20.
					tmp1S32 = tmp2S32 >> 14

					// Q20 / Q7 = Q13.
					if tmp1S32 > 0 {
						tmpS16 = int16(divW32W16(tmp1S32, nsk))
					} else {
						tmpS16 = int16(divW32W16(-tmp1S32, nsk))
						tmpS16 = -tmpS16
					}
					tmpS16 += 32       // Rounding
					nsk += tmpS16 >> 6 // Q13 >> 6 = Q7.
					if nsk < kMinStd {
						nsk = kMinStd
					}
					self.noiseStds[gaussian] = nsk
				}
			}

			// Separate models if they are too close.
			// noiseGlobalMean in Q14 (= Q7 * Q7).
			noiseGlobalMean = weightedAverage(self.noiseMeans[:], channel, 0,
				kNoiseDataWeights[:])
			// speechGlobalMean in Q14 (= Q7 * Q7).
			speechGlobalMean = weightedAverage(self.speechMeans[:], channel, 0,
				kSpeechDataWeights[:])

			// diff = "global" speech mean - "global" noise mean.
			// Q14 >> 9 - Q14 >> 9 = Q5.
			diff = int16(speechGlobalMean>>9) - int16(noiseGlobalMean>>9)
			if diff < kMinimumDifference[channel] {
				tmpS16 = kMinimumDifference[channel] - diff

				// tmp1S16 = ~0.8 * (kMinimumDifference - diff) in Q7.
				// tmp2S16 = ~0.2 * (kMinimumDifference - diff) in Q7.
				tmp1S16 = int16((13 * int32(tmpS16)) >> 2)
				tmp2S16 = int16((3 * int32(tmpS16)) >> 2)

				// Move Gaussian means for speech model by tmp1S16.
				speechGlobalMean = weightedAverage(self.speechMeans[:], channel,
					tmp1S16, kSpeechDataWeights[:])
				// Move Gaussian means for noise model by -tmp2S16.
				noiseGlobalMean = weightedAverage(self.noiseMeans[:], channel,
					-tmp2S16, kNoiseDataWeights[:])
			}

			// Control that the speech & noise means do not drift too much.
			maxspe = kMaximumSpeech[channel]
			tmp2S16 = int16(speechGlobalMean >> 7)
			if tmp2S16 > maxspe {
				tmp2S16 -= maxspe
				for k := 0; k < kNumGaussians; k++ {
					self.speechMeans[channel+k*kNumChannels] -= tmp2S16
				}
			}

			tmp2S16 = int16(noiseGlobalMean >> 7)
			if tmp2S16 > kMaximumNoise[channel] {
				tmp2S16 -= kMaximumNoise[channel]
				for k := 0; k < kNumGaussians; k++ {
					self.noiseMeans[channel+k*kNumChannels] -= tmp2S16
				}
			}
		}
		self.frameCounter++
	}

	// Smooth with respect to transition hysteresis.
	if vadflag == 0 {
		if self.overHang > 0 {
			vadflag = 2 + self.overHang
			self.overHang--
		}
		self.numOfSpeech = 0
	} else {
		self.numOfSpeech++
		if self.numOfSpeech > kMaxSpeechFrames {
			self.numOfSpeech = kMaxSpeechFrames
			self.overHang = overhead2
		} else {
			self.overHang = overhead1
		}
	}
	return vadflag
}

// initCore initializes the VAD core.
func initCore(self *vadInstT) int {
	// Initialization of general struct variables.
	self.vad = 1 // Speech active (=1).
	self.frameCounter = 0
	self.overHang = 0
	self.numOfSpeech = 0

	// Initialization of downsampling filter state.
	for i := range self.downsamplingFilterStates {
		self.downsamplingFilterStates[i] = 0
	}

	// Initialization of 48 to 8 kHz downsampling.
	resetResample48khzTo8khz(&self.state48To8)

	// Read initial PDF parameters.
	for i := 0; i < kTableSize; i++ {
		self.noiseMeans[i] = kNoiseDataMeans[i]
		self.speechMeans[i] = kSpeechDataMeans[i]
		self.noiseStds[i] = kNoiseDataStds[i]
		self.speechStds[i] = kSpeechDataStds[i]
	}

	// Initialize Index and Minimum value vectors.
	for i := 0; i < 16*kNumChannels; i++ {
		self.lowValueVector[i] = 10000
		self.indexVector[i] = 0
	}

	// Initialize splitting filter states.
	for i := range self.upperState {
		self.upperState[i] = 0
	}
	for i := range self.lowerState {
		self.lowerState[i] = 0
	}

	// Initialize high pass filter states.
	for i := range self.hpFilterState {
		self.hpFilterState[i] = 0
	}

	// Initialize mean value memory.
	for i := 0; i < kNumChannels; i++ {
		self.meanValue[i] = 1600
	}

	// Set aggressiveness mode to default.
	if setModeCore(self, kDefaultMode) != 0 {
		return -1
	}

	self.initFlag = kInitCheck

	return 0
}

// setModeCore sets the aggressiveness mode.
func setModeCore(self *vadInstT, mode int) int {
	switch mode {
	case 0:
		copy(self.overHangMax1[:], kOverHangMax1Q[:])
		copy(self.overHangMax2[:], kOverHangMax2Q[:])
		copy(self.individual[:], kLocalThresholdQ[:])
		copy(self.total[:], kGlobalThresholdQ[:])
	case 1:
		copy(self.overHangMax1[:], kOverHangMax1LBR[:])
		copy(self.overHangMax2[:], kOverHangMax2LBR[:])
		copy(self.individual[:], kLocalThresholdLBR[:])
		copy(self.total[:], kGlobalThresholdLBR[:])
	case 2:
		copy(self.overHangMax1[:], kOverHangMax1AGG[:])
		copy(self.overHangMax2[:], kOverHangMax2AGG[:])
		copy(self.individual[:], kLocalThresholdAGG[:])
		copy(self.total[:], kGlobalThresholdAGG[:])
	case 3:
		copy(self.overHangMax1[:], kOverHangMax1VAG[:])
		copy(self.overHangMax2[:], kOverHangMax2VAG[:])
		copy(self.individual[:], kLocalThresholdVAG[:])
		copy(self.total[:], kGlobalThresholdVAG[:])
	default:
		return -1
	}
	return 0
}

// calcVad8khz calculates VAD decision for 8 kHz input.
func calcVad8khz(inst *vadInstT, speechFrame []int16, frameLength int) int {
	var featureVector [kNumChannels]int16

	totalPower := calculateFeatures(inst, speechFrame, frameLength, featureVector[:])

	inst.vad = int(gmmProbability(inst, featureVector[:], totalPower, frameLength))

	return inst.vad
}

// calcVad16khz calculates VAD decision for 16 kHz input.
// Downsamples 16->8 kHz, then delegates to calcVad8khz.
func calcVad16khz(inst *vadInstT, speechFrame []int16, frameLength int) int {
	var speechNB [240]int16 // 30ms in 8kHz

	// Downsample 16->8 kHz.
	downsampling(speechFrame, speechNB[:], inst.downsamplingFilterStates[:])

	n := frameLength / 2
	return calcVad8khz(inst, speechNB[:n], n)
}

// calcVad32khz calculates VAD decision for 32 kHz input.
// Downsamples 32->16->8 kHz, then delegates to calcVad8khz.
func calcVad32khz(inst *vadInstT, speechFrame []int16, frameLength int) int {
	var speechWB [480]int16 // 30ms in 16kHz
	var speechNB [240]int16 // 30ms in 8kHz

	// Downsample 32->16 kHz.
	downsampling(speechFrame, speechWB[:], inst.downsamplingFilterStates[2:])
	n := frameLength / 2

	// Downsample 16->8 kHz.
	downsampling(speechWB[:n], speechNB[:], inst.downsamplingFilterStates[:])
	n /= 2

	return calcVad8khz(inst, speechNB[:n], n)
}

// calcVad48khz calculates VAD decision for 48 kHz input.
// Resamples 48->8 kHz, then delegates to calcVad8khz.
func calcVad48khz(inst *vadInstT, speechFrame []int16, frameLength int) int {
	var speechNB [240]int16 // 30 ms in 8 kHz.
	// tmpmem: frame length in 10 ms (480 samples) + 256 extra.
	tmpmem := make([]int32, 480+256)

	kFrameLen10ms48khz := 480
	kFrameLen10ms8khz := 80
	num10msFrames := frameLength / kFrameLen10ms48khz

	for i := 0; i < num10msFrames; i++ {
		// Note: C code always passes speech_frame (never advances) —
		// re-processes same data with evolving filter state.
		resample48khzTo8khz(
			speechFrame,
			speechNB[i*kFrameLen10ms8khz:(i+1)*kFrameLen10ms8khz],
			&inst.state48To8,
			tmpmem,
		)
	}

	return calcVad8khz(inst, speechNB[:frameLength/6], frameLength/6)
}
