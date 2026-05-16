package vad

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestAgainstCReference compares Go VAD decisions against C libfvad output frame-by-frame.
func TestAgainstCReference(t *testing.T) {
	wavFiles := []string{
		"audio_tiny8.wav",
		"audio_tiny16.wav",
		"audio_tiny32.wav",
		"audio_tiny48.wav",
	}
	modes := []int{0, 1, 2, 3}
	frameMsList := []int{10, 20, 30}

	totalErrors := 0

	for _, file := range wavFiles {
		wavPath := "testdata/" + file
		wav, err := readWav(wavPath)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", file, err)
		}

		for _, mode := range modes {
			for _, frameMs := range frameMsList {
				base := strings.TrimSuffix(file, ".wav")
				refPath := fmt.Sprintf("refout/%s_m%d_%dms.txt", base, mode, frameMs)

				refData, err := readRefFile(refPath)
				if err != nil {
					t.Fatalf("Failed to read ref %s: %v", refPath, err)
				}

				v := New()
				v.SetMode(Mode(mode))
				v.SetSampleRate(SampleRate(wav.sampleRate))

				frameSize := wav.sampleRate * frameMs / 1000
				if frameSize <= 0 || frameSize > len(wav.data) {
					t.Logf("Skipping %s mode=%d %dms: invalid frame size", file, mode, frameMs)
					continue
				}

				goResults := make([]int, 0, len(refData.results))
				for offset := 0; offset+frameSize <= len(wav.data); offset += frameSize {
					result, err := v.Process(wav.data[offset : offset+frameSize])
					if err != nil {
						t.Fatalf("%s mode=%d %dms: Process error at offset %d: %v", file, mode, frameMs, offset, err)
					}
					goResults = append(goResults, int(result))
				}

				if len(goResults) != len(refData.results) {
					t.Errorf("%s mode=%d %dms: Go=%d frames, C=%d frames",
						file, mode, frameMs, len(goResults), len(refData.results))
					totalErrors++
					continue
				}

				firstMismatch := -1
				mismatches := 0
				for i := 0; i < len(goResults); i++ {
					if goResults[i] != refData.results[i] {
						if firstMismatch < 0 {
							firstMismatch = i
						}
						mismatches++
					}
				}

				if mismatches > 0 {
					t.Errorf("%s mode=%d %dms: %d/%d frames differ (first at frame %d)",
						file, mode, frameMs, mismatches, len(goResults), firstMismatch)
					totalErrors++
				}
			}
		}
	}

	if totalErrors > 0 {
		t.Logf("Total: %d combinations have frame-level differences (out of %d)",
			totalErrors, len(wavFiles)*len(modes)*len(frameMsList))
	}
}

type refData struct {
	results []int // per-frame VAD results (0 or 1)
}

func readRefFile(path string) (*refData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rd := &refData{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		v, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			continue
		}
		rd.results = append(rd.results, v)
	}
	return rd, scanner.Err()
}
