// Copyright (c) 2023 Alexander Khudich
//
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
	"testing"
)

func BenchmarkStreaming(b *testing.B) {
	const accel = 1.2
	var samplesPerFrame int

	pcm, sampleRate, channels, err := readWAV("./testdata/OSR_us_000_0010_8k.wav")
	if err != nil {
		b.Fatalf("reading error: %v", err)
	}

	samplesPerFrame = sampleRate * channels * 20 / 1000

	var in [][]int16
	for i := 0; samplesPerFrame < len(pcm)-i; i += samplesPerFrame {
		in = append(in, pcm[i:i+samplesPerFrame])
	}

	buf := make([]int16, samplesPerFrame)

	b.Run("without sonic", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, data := range in {
				copy(buf, data) // Emulate decoding
			}
		}
	})

	b.Run("with sonic", func(b *testing.B) {
		sonicStream := NewSonicStream(sampleRate, channels)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, data := range in {
				copy(buf, data) // Emulate decoding
				sonicStream.Write(buf)
				sonicStream.ReadTo(buf)
			}
		}
	})

	b.Run("with sonic accelerated", func(b *testing.B) {
		var empty int

		sonicStream := NewSonicStream(sampleRate, channels)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for j, data := range in {
				if j%100 == 50 {
					sonicStream.SetSpeed(accel)
				} else if j%100 == 75 {
					sonicStream.SetSpeed(1)
				}
				copy(data, buf) // Emulate decoding
				sonicStream.Write(buf)

				if sonicStream.NumOutputSamples() >= samplesPerFrame {
					sonicStream.ReadTo(buf)
				} else {
					empty++
				}
			}
		}
	})

	b.Run("with sonic streaming", func(b *testing.B) {
		var empty int

		sonicStream := NewZeroCopyStream(sampleRate, channels)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for _, data := range in {
				tempAudioBuf, err := sonicStream.Process(samplesPerFrame, func(buf []int16) error {
					copy(buf, data) // Emulate decoding
					return nil
				})

				if err != nil {
					b.Fatal("can't read filled frame: " + err.Error())
				}

				if len(tempAudioBuf) == 0 {
					empty++
				}
			}
		}
	})

	b.Run("with sonic streaming accelerated", func(b *testing.B) {
		var empty int

		sonicStream := NewZeroCopyStream(sampleRate, channels)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			sonicStream.Reset()
			for j, data := range in {
				if j%100 == 50 {
					sonicStream.SetSpeed(accel)
				} else if j%100 == 75 {
					sonicStream.SetSpeed(1)
				}

				tempAudioBuf, err := sonicStream.Process(samplesPerFrame, func(buf []int16) error {
					copy(buf, data) // Emulate decoding
					return nil
				})

				if err != nil {
					b.Fatal("can't read filled frame: " + err.Error())
				}

				if len(tempAudioBuf) == 0 {
					empty++
				}
			}
		}
	})
}
