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

import "fmt"

type ZeroCopyStream struct {
	*Sonic
}

// NewZeroCopyStream creates a new sonic NonCopy Sonic.
func NewZeroCopyStream(sampleRate, numChannels int) *ZeroCopyStream {
	return &ZeroCopyStream{NewSonic(sampleRate, numChannels)}
}

// BorrowRawSlice (EXPERIMENTAL) borrows Raw slice from the sonic's input buffer for direct use, for example as audio decoding
// target memory.
// This slice must not be moved in case you want to return it back to the input buffer.
func (s *ZeroCopyStream) BorrowRawSlice(n int) []int16 {
	return s.inputBuffer.RawSlice(n)
}

// ReturnRawSlice (EXPERIMENTAL) assumes that len(s) bytes are filled with audio data and adds up internal counters
// to include audio data from s into internal buffer space.
// This function should be called right after corresponding BorrowRawSlice. You can't borrow several slices and then
// return it in bulk manner
func (s *ZeroCopyStream) ReturnRawSlice(slice []int16) error {
	samples := len(slice) / s.numChannels
	if err := s.inputBuffer.RawLenAdd(samples); err != nil {
		return err
	}
	s.inputPlaytime = float64(s.inputSamplesLen()) * s.samplePeriod / (s.speed / s.pitch)
	return nil
}

func (s *ZeroCopyStream) Process(size int, f func(buf []int16) error) ([]int16, error) {
	tempAudioBuf := s.BorrowRawSlice(size)

	if err := f(tempAudioBuf); err != nil {
		return nil, fmt.Errorf("function call: %w", err)
	}
	if err := s.ReturnRawSlice(tempAudioBuf); err != nil {
		return nil, fmt.Errorf("buffer return: %w", err)
	}

	data, err := s.Read(size)
	if err != nil {
		return nil, fmt.Errorf("s reading: %w", err)
	}

	return data, nil
}

// read (EXPERIMENTAL) returns slice directly from input buffer if there is no any audio changes were expected (no speed, pitch or volume changes)
// Otherwise it works as usual - process ausio from input buffer to output and returns the slice from output buffer.
func (s *ZeroCopyStream) read(num int) ([]int16, error) {
	var data []int16
	var err error

	samples := num / s.numChannels
	if samples == 0 {
		return data, err
	}

	iLen := s.inputBuffer.Len()
	oLen := s.outputBuffer.Len()

	rate := s.rate * s.pitch
	speed := float64(iLen) * s.samplePeriod / s.inputPlaytime

	if speed > 0.99999 && speed < 1.00001 && rate == 1 && s.volume == 1.0 {
		if iLen >= samples || oLen >= samples {
			switch {
			case oLen == 0:
				data, err = s.inputBuffer.ReadSlice(samples)
				s.inputPlaytime = float64(s.inputSamplesLen()) * s.samplePeriod / (s.speed / s.pitch)
			case oLen >= samples:
				data, err = s.outputBuffer.ReadSlice(samples)
			default:
				if iLen >= oLen {
					data, err = s.inputBuffer.ReadSlice(samples)
					odata, _ := s.outputBuffer.ReadSlice(samples)
					crossFade(data, odata)

					s.inputPlaytime = float64(s.inputSamplesLen()) * s.samplePeriod / (s.speed / s.pitch)
				}
			}
		}
	} else {
		if err := s.processStreamInput(); err != nil {
			return nil, err
		} else if s.outputBuffer.Len() >= samples {
			data, err = s.outputBuffer.ReadSlice(samples)
		}
	}

	return data, err
}

func (s *ZeroCopyStream) Read(num int) ([]int16, error) {
	return s.read(num)
}

func (s *ZeroCopyStream) ReadTo(to []int16) ([]int16, error) {
	data, err := s.read(cap(to))
	if err != nil {
		return to[:0], err
	}
	to = to[:len(data)]
	copy(to, data)

	return to, nil
}

// crossFade crossfades buf with tail. buf length must be greater or equal to tail length
func crossFade(buf, tail []int16) {
	l := len(tail)
	for i, decrV := range tail {
		incrV := buf[i]

		buf[i] = int16((int(decrV)*(l-i) + int(incrV)*i) / l)
	}
}
