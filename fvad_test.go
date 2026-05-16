package vad

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestNewAndReset(t *testing.T) {
	v := New()
	if v == nil {
		t.Fatal("New() returned nil")
	}
	if v.core.initFlag != kInitCheck {
		t.Errorf("initFlag = %d, want %d", v.core.initFlag, kInitCheck)
	}
	v.Reset()
	if v.core.initFlag != kInitCheck {
		t.Errorf("initFlag after reset = %d, want %d", v.core.initFlag, kInitCheck)
	}
}

func TestSetMode(t *testing.T) {
	v := New()
	if err := v.SetMode(Mode(-1)); err == nil {
		t.Error("SetMode(-1) should fail")
	}
	if err := v.SetMode(Mode(4)); err == nil {
		t.Error("SetMode(4) should fail")
	}
	for _, mode := range []Mode{ModeQuality, ModeLowBitrate, ModeAggressive, ModeVeryAggressive} {
		if err := v.SetMode(mode); err != nil {
			t.Errorf("SetMode(%d) = %v", mode, err)
		}
	}
}

func TestSetSampleRate(t *testing.T) {
	v := New()
	if err := v.SetSampleRate(SampleRate(9999)); err == nil {
		t.Error("SetSampleRate(9999) should fail")
	}
	for _, sr := range []SampleRate{SampleRate8k, SampleRate16k, SampleRate32k, SampleRate48k} {
		if err := v.SetSampleRate(sr); err != nil {
			t.Errorf("SetSampleRate(%d) = %v", sr, err)
		}
	}
}

func TestProcessZeros(t *testing.T) {
	v := New()
	v.SetSampleRate(SampleRate8k)
	zeros := make([]int16, 80)
	result, err := v.Process(zeros)
	if err != nil {
		t.Fatal(err)
	}
	if result != ResultNoVoice {
		t.Errorf("Process(zeros) = %d, want ResultNoVoice", result)
	}
}

func TestProcessInvalidLength(t *testing.T) {
	v := New()
	v.SetSampleRate(SampleRate8k)
	_, err := v.Process(make([]int16, 100))
	if err == nil {
		t.Error("Process(invalid length) should fail")
	}
}

func TestProcessSpeech(t *testing.T) {
	v := New()
	v.SetSampleRate(SampleRate8k)
	speech := make([]int16, 80)
	for i := 0; i < 80; i++ {
		speech[i] = int16(i * i)
	}
	result, err := v.Process(speech)
	if err != nil {
		t.Fatal(err)
	}
	if result != ResultVoice {
		t.Errorf("Process(speech) = %d, want ResultVoice", result)
	}
}

type wavReader struct {
	sampleRate int
	data       []int16
}

func readWav(path string) (*wavReader, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) < 44 {
		return nil, fmt.Errorf("file too small for WAV header")
	}
	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a WAVE file")
	}
	var result wavReader
	pos := 12
	for pos+8 <= len(data) {
		chunkID := string(data[pos : pos+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		pos += 8
		if pos+chunkSize > len(data) {
			break
		}
		switch chunkID {
		case "fmt ":
			audioFormat := binary.LittleEndian.Uint16(data[pos : pos+2])
			if audioFormat != 1 {
				return nil, fmt.Errorf("not PCM")
			}
			channels := binary.LittleEndian.Uint16(data[pos+2 : pos+4])
			if channels != 1 {
				return nil, fmt.Errorf("only mono supported")
			}
			result.sampleRate = int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		case "data":
			numSamples := chunkSize / 2
			result.data = make([]int16, numSamples)
			for i := 0; i < numSamples; i++ {
				result.data[i] = int16(binary.LittleEndian.Uint16(data[pos+i*2 : pos+i*2+2]))
			}
		}
		pos += chunkSize
	}
	if result.data == nil {
		return nil, fmt.Errorf("no data chunk found")
	}
	return &result, nil
}

type expectedResult struct {
	file        string
	mode        int
	frameMs     int
	voiceFrames int
	totalFrames int
}

func parseExpected(path string) ([]expectedResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var results []expectedResult
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "===") {
			continue
		}
		line = strings.Trim(line, "= ")
		parts := strings.Split(line, ";")
		if len(parts) < 3 {
			continue
		}
		filePart := strings.TrimSpace(parts[0])
		modeStr := strings.TrimPrefix(strings.TrimSpace(parts[1]), "mode ")
		mode, _ := strconv.Atoi(modeStr)
		msStr := strings.TrimSuffix(strings.TrimSpace(parts[2]), " ms")
		frameMs, _ := strconv.Atoi(msStr)

		if !scanner.Scan() {
			break
		}
		er := expectedResult{file: filePart, mode: mode, frameMs: frameMs}
		fmt.Sscanf(scanner.Text(), "voice detected in %d of %d frames", &er.voiceFrames, &er.totalFrames)
		results = append(results, er)
		scanner.Scan() // skip segment line 1
		scanner.Scan() // skip segment line 2
	}
	return results, scanner.Err()
}

func TestWavRegression(t *testing.T) {
	expected, err := parseExpected("testdata/wavtest.expect")
	if err != nil {
		t.Fatal("Failed to parse expected output:", err)
	}
	if len(expected) == 0 {
		t.Fatal("No expected results parsed")
	}

	wavFiles := map[string]string{
		"audio_tiny8.wav":  "testdata/audio_tiny8.wav",
		"audio_tiny16.wav": "testdata/audio_tiny16.wav",
		"audio_tiny32.wav": "testdata/audio_tiny32.wav",
		"audio_tiny48.wav": "testdata/audio_tiny48.wav",
	}

	failures := 0
	for _, exp := range expected {
		wavPath, ok := wavFiles[exp.file]
		if !ok {
			continue
		}
		wav, err := readWav(wavPath)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", exp.file, err)
		}

		v := New()
		v.SetMode(Mode(exp.mode))
		v.SetSampleRate(SampleRate(wav.sampleRate))

		frameSize := wav.sampleRate * exp.frameMs / 1000
		if frameSize <= 0 || frameSize > len(wav.data) {
			continue
		}

		voiceCount := 0
		totalFrames := 0
		for offset := 0; offset+frameSize <= len(wav.data); offset += frameSize {
			result, err := v.Process(wav.data[offset : offset+frameSize])
			if err != nil {
				t.Fatalf("%s mode=%d %dms: Process error: %v", exp.file, exp.mode, exp.frameMs, err)
			}
			if result == ResultVoice {
				voiceCount++
			}
			totalFrames++
		}

		if totalFrames != exp.totalFrames {
			t.Errorf("%s mode=%d %dms: %d frames, want %d", exp.file, exp.mode, exp.frameMs, totalFrames, exp.totalFrames)
			failures++
			continue
		}

		diff := voiceCount - exp.voiceFrames
		if diff < 0 {
			diff = -diff
		}
		tolerance := exp.totalFrames * 5 / 100
		if diff > tolerance {
			t.Errorf("%s mode=%d %dms: voice %d/%d, want %d/%d (diff=%d > tolerance=%d)",
				exp.file, exp.mode, exp.frameMs, voiceCount, totalFrames,
				exp.voiceFrames, exp.totalFrames, diff, tolerance)
			failures++
		}
	}

	if failures > 0 {
		t.Logf("Note: %d regression mismatches. Resampler paths may need tuning for bit-exact match.", failures)
	}
}

func TestMultipleInstances(t *testing.T) {
	v1 := New()
	v2 := New()
	v1.SetSampleRate(SampleRate8k)
	v2.SetSampleRate(SampleRate8k)
	speech := make([]int16, 80)
	for i := 0; i < 80; i++ {
		speech[i] = int16(i * i)
	}
	for i := 0; i < 10; i++ {
		r1, _ := v1.Process(speech)
		r2, _ := v2.Process(speech)
		if r1 != r2 {
			t.Errorf("iteration %d: v1=%d, v2=%d", i, r1, r2)
		}
	}
}

func BenchmarkProcess8k(b *testing.B) {
	v := New()
	v.SetSampleRate(SampleRate8k)
	frame := make([]int16, 160)
	for i := range frame {
		frame[i] = int16(i * i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Process(frame)
	}
}
