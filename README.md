# go-vad

[![Go Reference](https://pkg.go.dev/badge/github.com/rolandhe/go-vad.svg)](https://pkg.go.dev/github.com/rolandhe/go-vad)
[中文说明](README_CN.md)

Pure Go Voice Activity Detection (VAD) library — a faithful port of [libfvad](https://github.com/dpirch/libfvad), which is the standalone VAD engine extracted from the [WebRTC](https://webrtc.org/) native code package.

**Zero external dependencies. No cgo.** Bit-exact output verified against the C reference implementation across all sample rates, modes, and frame lengths.

## Installation

```bash
go get github.com/rolandhe/go-vad
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/rolandhe/go-vad"
)

func main() {
    // Create a new VAD instance.
    v := vad.New()

    // Configure: 16 kHz sample rate, aggressive mode.
    v.SetSampleRate(vad.SampleRate16k)
    v.SetMode(vad.ModeAggressive)

    // Process a 20ms frame (320 samples at 16 kHz).
    frame := make([]int16, 320)
    // ... fill frame with 16-bit PCM audio data ...

    result, err := v.Process(frame)
    if err != nil {
        panic(err)
    }
    if result == vad.ResultVoice {
        fmt.Println("Speech detected")
    } else {
        fmt.Println("Silence / non-speech")
    }
}
```

## API Reference

### Types

```go
type Mode int
```

VAD aggressiveness mode. Higher modes are more restrictive — they classify fewer frames as speech, reducing false positives but increasing missed detections.

| Constant | Value | Description |
|----------|-------|-------------|
| `ModeQuality` | `0` | Normal quality (default) |
| `ModeLowBitrate` | `1` | Low bitrate |
| `ModeAggressive` | `2` | Aggressive |
| `ModeVeryAggressive` | `3` | Very aggressive |

```go
type SampleRate int
```

Valid input sample rate in Hz.

| Constant | Value |
|----------|-------|
| `SampleRate8k` | `8000` |
| `SampleRate16k` | `16000` |
| `SampleRate32k` | `32000` |
| `SampleRate48k` | `48000` |

```go
type Result int
```

Per-frame VAD decision.

| Constant | Value | Description |
|----------|-------|-------------|
| `ResultNoVoice` | `0` | No active voice detected |
| `ResultVoice` | `1` | Active voice detected |

### Constructor

```go
func New() *VAD
```

Creates and initializes a new VAD instance with default settings:
- Mode: `ModeQuality` (0)
- Sample rate: `SampleRate8k` (8000 Hz)

### Methods

```go
func (v *VAD) SetMode(mode Mode) error
```

Sets the aggressiveness mode. Returns an error if mode is not 0–3.

```go
func (v *VAD) SetSampleRate(sr SampleRate) error
```

Sets the input sample rate. Valid values: `8000`, `16000`, `32000`, `48000`. Returns an error for invalid rates. Internally, all processing is done at 8 kHz — higher rates are downsampled first.

```go
func (v *VAD) Process(frame []int16) (Result, error)
```

Processes one frame of 16-bit signed PCM audio and returns the VAD decision.

The frame length must correspond to **10 ms, 20 ms, or 30 ms** at the configured sample rate:

| Sample Rate | 10 ms | 20 ms | 30 ms |
|-------------|-------|-------|-------|
| 8000 Hz | 80 | 160 | 240 |
| 16000 Hz | 160 | 320 | 480 |
| 32000 Hz | 320 | 640 | 960 |
| 48000 Hz | 480 | 960 | 1440 |

Returns an error if the frame length is invalid.

```go
func (v *VAD) Reset()
```

Reinitializes the VAD instance, clearing all internal state and resetting mode and sample rate to defaults.

## Frame Size Reference

Choose the frame duration that matches your application's requirements:

| Duration | Latency | Sensitivity |
|----------|---------|-------------|
| 10 ms | Lowest | Good for short utterances |
| 20 ms | Medium | Balanced |
| 30 ms | Higher | Best frequency resolution |

## Concurrency

Each `*VAD` instance is **stateful** and not safe for concurrent use from multiple goroutines. If you need to process multiple audio streams concurrently, create a separate `*VAD` instance per stream.

The library itself has no global mutable state and is safe for concurrent instantiation.

## Algorithm Overview

The VAD operates on 8 kHz audio and works as follows:

1. **Downsampling**: Input at >8 kHz is downsampled to 8 kHz using allpass-filter-based decimators.
2. **Feature extraction**: A quadrature mirror filter (QMF) bank splits the 0–4 kHz spectrum into 6 sub-bands (80–250, 250–500, 500–1000, 1000–2000, 2000–3000, 3000–4000 Hz). The log-energy of each band forms a 6-dimensional feature vector.
3. **GMM scoring**: A Gaussian Mixture Model (2 Gaussians × 6 channels) computes the likelihood of the feature vector under speech and noise hypotheses.
4. **Decision**: Local (per-band) and global log-likelihood ratio tests are made against mode-dependent thresholds. The GMM parameters adapt online to the input signal.
5. **Hangover smoothing**: When speech ends, the VAD continues to report speech for a brief period to avoid mid-word cutoffs.

All arithmetic is **fixed-point** (Q-format), exactly matching the WebRTC reference implementation.

## License

BSD 3-Clause. See the [LICENSE](LICENSE) file for details. Based on WebRTC code copyright The WebRTC project authors, and libfvad copyright Daniel Pirch.
