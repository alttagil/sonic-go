// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package sonic

import (
	"fmt"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"log"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
	"unsafe"
)

func TestSpeed(t *testing.T) {
	w, sampleRate, channels, err := readWAV("./testdata/OSR_us_000_0010_8k.wav")
	if err != nil {
		t.Fatalf("reading error: %v", err)
	}

	startTime := time.Now()
	samples, err := ChangeSpeed(sampleRate, channels, 1.5, 1, 1, 1, w)
	elapsedTime := time.Since(startTime)
	log.Println("Elapsed:", elapsedTime)

	if err != nil {
		t.Fatalf("reading error: %v", err)
	}

	_ = dumpPCM(sampleRate, channels, samples, "./testdata/out.wav")
}

func TestSpeed2(t *testing.T) {
	w, sampleRate, channels, err := readWAV("./testdata/OSR_us_000_0010_8k.wav")
	if err != nil {
		t.Fatalf("reading error: %v", err)
	}

	stream := NewSonicStream(sampleRate, channels)
	stream.SetSpeed(1.5)
	stream.SetPitch(1)
	stream.SetRate(1)
	stream.SetVolume(1)
	if err := stream.AddSamples(w); err != nil {
		t.Fatal(err)
	}
	startTime := time.Now()
	if err := stream.processStreamInput(); err != nil {
		t.Fatal(err)
	}
	elapsedTime := time.Since(startTime)
	log.Println("Elapsed2:", elapsedTime)

	if err != nil {
		t.Fatalf("reading error: %v", err)
	}
}

func BenchmarkSonic(b *testing.B) {
	w, sampleRate, channels, err := readWAV("./testdata/OSR_us_000_0010_8k.wav")
	if err != nil {
		b.Fatalf("reading error: %v", err)
	}
	stream := NewSonicStream(sampleRate, channels)
	stream.SetSpeed(1.5)
	stream.SetPitch(1)
	stream.SetRate(1)
	stream.SetVolume(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := stream.AddSamples(w); err != nil {
			b.Fatal(err)
		}
		if err := stream.processStreamInput(); err != nil {
			b.Fatal(err)
		}
		if err := stream.Flush(); err != nil {
			b.Fatal(err)
		}
		if _, err := stream.ReadAll(); err != nil {
			b.Fatal(err)
		}
	}

}

func BenchmarkSonicOnlyProcess(b *testing.B) {
	w, sampleRate, channels, err := readWAV("./testdata/OSR_us_000_0010_8k.wav")
	if err != nil {
		b.Fatalf("reading error: %v", err)
	}
	stream := NewSonicStream(sampleRate, channels)
	stream.SetSpeed(1.5)
	stream.SetPitch(1)
	stream.SetRate(1)
	stream.SetVolume(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := stream.AddSamples(w); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if err := stream.processStreamInput(); err != nil {
			b.Fatal(err)
		}
		b.StopTimer()
		if err := stream.Flush(); err != nil {
			b.Fatal(err)
		}
		if _, err := stream.ReadAll(); err != nil {
			b.Fatal(err)
		}
	}

}

func readWAV(fname string) ([]int16, int, int, error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, 0, 0, err
	}
	defer f.Close()

	buf, err := wav.NewDecoder(f).FullPCMBuffer()

	if err != nil {
		return nil, 0, 0, err
	}
	// if buf.SourceBitDepth != 16 || buf.Format.sampleRate != 48000 || buf.Format.numChannels != 1 {
	if buf.SourceBitDepth != 16 {
		return nil, 0, 0, fmt.Errorf("invalid format(bit_depth=%v, sample_rate=%v, channels=%v)",
			buf.SourceBitDepth, buf.Format.SampleRate, buf.Format.NumChannels)
	}
	out := make([]int16, len(buf.Data))
	for i := range out {
		out[i] = int16(buf.Data[i])
	}
	return out, buf.Format.SampleRate, buf.Format.NumChannels, nil
}

func dumpPCM(samplerate, channels int, pcm []int16, filename string) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0750); err != nil {
		return err
	}
	dumpInFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer dumpInFile.Close()
	enc := wav.NewEncoder(dumpInFile, samplerate, 16, channels, 1)
	defer enc.Close()

	f := &audio.Format{
		NumChannels: channels,
		SampleRate:  samplerate,
	}

	samples := make([]int, len(pcm))
	for i := range pcm {
		samples[i] = int(pcm[i])
	}

	if err := enc.Write(&audio.IntBuffer{
		Format:         f,
		SourceBitDepth: 16,
		Data:           samples,
	}); err != nil {
		return err
	}
	return nil
}

func BenchmarkFindPitchPeriod(b *testing.B) {
	Period := []int16{-16, -15, -13, -15, -16, -17, -16, -16, -14, -11, -9, -9, -9, -6, -9, -10, -9, -8, -5, -5, -6,
		-11, -14, -11, -9, -8, -7, -8, -11, -14, -16, -17, -19, -18, -14, -12, -10, -7, -8, -14, -16, -11, -7, -4, -3,
		-2, -6, -14, -17, -21, -25, -26, -25, -27, -31, -33, -31, -26, -21, -20, -20, -22, -26, -28, -29, -31, -30,
		-28, -28, -27, -23, -23, -24, -25, -26, -24, -22, -18, -10, -2, -2, -4, -5, -8, -14, -17, -15, -16, -21, -23,
		-21, -20, -23, -25, -23, -20, -19, -17, -14, -9, -10, -17, -24, -25, -26, -31, -29, -23, -21, -21, -15, -6,
		-1, 3, 3, 2, 6, 11, 13, 13, 10, 9, 11, 10, 5, -2, -6, -6, -6, -7, -7, -7, -7, -8, -9, -10, -10, -13, -17, -18,
		-21, -21, -17, -15, -19, -17, -13, -16, -19, -15, -20, -27, -23, -13, -5, -6, -6, -1, -2, -6, -8, -9, -12, -17,
		-22, -22, -23, -24, -22, -20, -16, -9, 0, 10, 12, 7, 4, 0, -8, -12, -15, -18, -17, -16, -10, -11, -20, -27,
		-28, -22, -18, -15, -7, -1, 0, 3, 9, 9, 2, -2, -6, -12, -11, -2, 3, 0, -2, -7, -11, -7, -1, 3, 13, 21, 27,
		31, 33, 32, 32, 31, 26, 20, 14, 8, 5, 7, 12, 19, 18, 11, 9, 4, -7, -16, -20, -25, -29, -29, -28, -32, -32,
		-26, -15, -8, -4, 8, 15, 13, 11, 10, 10, 10, 8, 10, 10, 12, 7, -2, -11, -14, -10, -5, -9, -13, -13, -12, -3,
		10, 10, -2, -1, 11, 17, 24, 30, 31, 23, 9, 1, -5, -14, -16, -7, 9, 21, 29, 29, 24, 30, 38, 40, 47, 56, 53, 47,
		42, 34, 22, 7, -6, -14, -14, -13, -17, -23, -27, -25, -23, -13, -9, -7, 7, 21, 32, 46, 56, 55, 53, 46, 37, 30,
		25, 27, 33, 37, 38, 38, 43, 51, 54, 58, 57, 48, 43, 48, 50, 44, 38, 31, 28, 24, 16, 12, 9, 19, 31, 38, 44, 36,
		22, 20, 21, 11, 4, 13, 21, 26, 31, 34, 31}

	B := NewSampleBuffer(1, 400)
	_ = B.WriteSlice(Period)

	b.Run("cgo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			findPitchPeriodInRange(B, 120, 180)
		}
	})
	b.Run("native", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			findPitchPeriodInRangeNative(B, 120, 180)
		}
	})
	b.Run("nativea", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			findPitchPeriodInRangeNativeA(B, 120, 180)
		}
	})
	b.Run("nativeabs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			findPitchPeriodInRangeNativeAbs(B, 120, 180)
		}
	})
	b.Run("nativeunsafe", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			findPitchPeriodInRangeNativeUnsafe(B, 120, 180)
		}
	})
}

func findPitchPeriodInRangeNativeA(b *SampleBuffer, minP, maxP int) (int, int, int) {
	var diff, minDiff, maxDiff int

	var bestPeriod int = 0
	var worstPeriod int = 255

	minDiff = 1
	maxDiff = 0
	samples, _ := b.GetSlice(2 * maxP)

	_ = samples[len(samples)-1]

	for period := minP; period <= maxP; period++ {
		diff = 0

		for i := 0; i < period; i++ {
			sVV := samples[i]
			pVV := samples[i+period]
			diff2 := int(sVV - pVV)
			diff2 = (diff2 + (diff2 >> 63)) ^ (diff2 >> 63)
			diff += diff2
		}

		if bestPeriod == 0 || diff*bestPeriod < minDiff*period {
			minDiff = diff
			bestPeriod = period
		}

		if diff*worstPeriod > maxDiff*period {
			maxDiff = diff
			worstPeriod = period
		}
	}

	return bestPeriod, minDiff / bestPeriod, maxDiff / worstPeriod
}

func findPitchPeriodInRangeNative(b *SampleBuffer, minP, maxP int) (int, int, int) {
	var diff, minDiff, maxDiff int

	var bestPeriod int = 0
	var worstPeriod int = 255

	minDiff = 1
	maxDiff = 0
	samples, _ := b.GetSlice(2 * maxP)

	_ = samples[len(samples)-1]

	for period := minP; period <= maxP; period++ {
		diff = 0

		for i := 0; i < period; i++ {
			sVV := samples[i]
			pVV := samples[i+period]

			if sVV >= pVV {
				diff += int(sVV - pVV)
			} else {
				diff += int(pVV - sVV)
			}
		}

		if bestPeriod == 0 || diff*bestPeriod < minDiff*period {
			minDiff = diff
			bestPeriod = period
		}

		if diff*worstPeriod > maxDiff*period {
			maxDiff = diff
			worstPeriod = period
		}
	}

	return bestPeriod, minDiff / bestPeriod, maxDiff / worstPeriod
}

func findPitchPeriodInRangeNativeAbs(b *SampleBuffer, minP, maxP int) (int, int, int) {
	var diff, minDiff, maxDiff int

	var bestPeriod int = 0
	var worstPeriod int = 255

	minDiff = 1
	maxDiff = 0
	samples, _ := b.GetSlice(2 * maxP)

	_ = samples[len(samples)-1]

	for period := minP; period <= maxP; period++ {
		diff = 0

		for i := 0; i < period; i++ {
			diff += int(math.Abs(float64(samples[i] - samples[i+period])))
		}

		if bestPeriod == 0 || diff*bestPeriod < minDiff*period {
			minDiff = diff
			bestPeriod = period
		}

		if diff*worstPeriod > maxDiff*period {
			maxDiff = diff
			worstPeriod = period
		}
	}

	return bestPeriod, minDiff / bestPeriod, maxDiff / worstPeriod
}

func findPitchPeriodInRangeNativeUnsafe(b *SampleBuffer, minP, maxP int) (int, int, int) {
	var diff, minDiff, maxDiff uint64

	bestPeriod := 0
	worstPeriod := 255
	minDiff = 1
	maxDiff = 0

	samples, _ := b.GetSlice(2 * maxP)

	unsafePointer := unsafe.Pointer(&samples[0])
	sizeOfInt := unsafe.Sizeof(samples[0])
	sP := uintptr(0)
	pP := uintptr(minP * int(sizeOfInt))

	for period := minP; period <= maxP; period++ {
		diff = 0
		for i := 0; i < period; i++ {
			sVV := *(*uint16)(unsafe.Pointer(uintptr(unsafePointer) + sP))
			pVV := *(*uint16)(unsafe.Pointer(uintptr(unsafePointer) + pP))

			sP += sizeOfInt
			pP += sizeOfInt

			if sVV >= pVV {
				diff += uint64(uint16(sVV - pVV))
			} else {
				diff += uint64(uint16(pVV - sVV))
			}
		}

		if bestPeriod == 0 || diff*uint64(bestPeriod) < minDiff*uint64(period) {
			minDiff = diff
			bestPeriod = period
		}

		if diff*uint64(worstPeriod) > maxDiff*uint64(period) {
			maxDiff = diff
			worstPeriod = period
		}
	}

	return bestPeriod, int(minDiff / uint64(bestPeriod)), int(maxDiff / uint64(worstPeriod))
}
