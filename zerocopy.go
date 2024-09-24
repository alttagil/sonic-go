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
	"fmt"
)

type ZeroCopyStream struct {
	*Sonic
}

// NewZeroCopyStream creates a new instance of ZeroCopyStream, which wraps a Sonic instance.
// It initializes the Sonic stream with the specified sample rate and number of audio channels.
func NewZeroCopyStream(sampleRate, numChannels int) *ZeroCopyStream {
	return &ZeroCopyStream{NewSonic(sampleRate, numChannels)}
}

// Process processes a specified number of `samples` (`numChannels` * `samples` bytes) from the Sonic buffer.
// It borrows a buffer slice from Sonic internal inputBuffer, allows external code (via function `f`)
// to fill the buffer with decoded audio samples, and then returns the buffer for processing.
// The function ensures the slice is returned back to the buffer and processes the samples before
// providing a result for further use.
//
// if there are no enough data in the buffers Process returns nil slice and nil error
//
// Pseudocode example:
//
//		tempAudioBuf, err := sonic.Process(frameSize, func(buf []int16) error {
//			if err := opus.decodePacket(buf, someOpusPacket); err != nil {
//				return fmt.Errorf("decode packet: %w", err)
//			}
//			return nil
//		})
//
//	 opus.encodePacket(tempAudioBuf, someOtherPacket)
//
// Note: The returned slice must be processed before the next Process call, as it may be overwritten
// during subsequent processing.
func (s *ZeroCopyStream) Process(samples int, f func(buf []int16) error) ([]int16, error) {
	// Borrow a buffer slice from the Sonic input buffer
	tempAudioBuf := s.borrowRawSlice(samples)

	// Call the provided function to fill the buffer with audio samples
	if err := f(tempAudioBuf); err != nil {
		return nil, fmt.Errorf("function call: %w", err)
	}

	// Return the borrowed buffer to Sonic after processing
	if err := s.returnRawSlice(tempAudioBuf); err != nil {
		return nil, fmt.Errorf("buffer return: %w", err)
	}

	// Borrows buffer from Sonic input buffer (if no changes made to the input data) or from output buffer
	data, err := s.processAndRead(samples)
	if err != nil {
		return nil, fmt.Errorf("s reading: %w", err)
	}

	return data, nil
}

// borrowRawSlice borrows a raw slice of size `n` from the Sonic input buffer.
// This slice can be used for direct operations such as audio decoding.
// Care must be taken to avoid moving the slice if it needs to be returned.
func (s *ZeroCopyStream) borrowRawSlice(n int) []int16 {
	return s.inputBuffer.RawSlice(n)
}

// returnRawSlice returns the borrowed slice back to the Sonic input buffer and updates the internal
// counters to reflect the new audio data.
// It is essential that this function is called immediately after borrowRawSlice. Borrowing multiple
// slices and returning them in bulk is not supported.
func (s *ZeroCopyStream) returnRawSlice(slice []int16) error {
	samples := len(slice) / s.numChannels
	if err := s.inputBuffer.RawLenAdd(samples); err != nil {
		return err
	}
	s.updateInputPlaytime()
	return nil
}

// processAndRead returns a slice from the Sonic input buffer, bypassing any audio changes if none are required
// (i.e., no speed, pitch, or volume adjustments).
// If changes are required, it processes the audio from the input buffer to the output buffer and returns
// a slice from the output buffer.
// if there are no enough data in the buffers processAndRead returns nil slice and nil error
func (s *ZeroCopyStream) processAndRead(samples int) (data []int16, err error) {
	if samples == 0 {
		return
	}

	if s.speed == 1.0 && s.erate == 1.0 && s.volume == 1.0 {
		iLen := s.inputBuffer.Len()
		oLen := s.outputBuffer.Len()

		switch {
		case iLen+oLen < samples:
			break
		case oLen == 0: // if there are no sped up samples in the output queue
			data, err = s.inputBuffer.ReadSlice(samples)
			s.updateInputPlaytime()
		case oLen >= samples: // if there are enough sped up samples in the output buffer
			data, err = s.outputBuffer.ReadSlice(samples)
		default:
			s.inputBuffer.MoveTo(s.outputBuffer, samples-oLen)
			data, _ = s.outputBuffer.ReadSlice(samples)
			s.updateInputPlaytime()
		}

	} else {
		if err = s.processStreamInput(); err != nil {
			return
		} else if s.outputBuffer.Len() >= samples {
			data, err = s.outputBuffer.ReadSlice(samples)
			return
		}
	}

	return
}

// read retrieves `num` samples from the Sonic buffer by invoking the internal `processAndRead` method.
// if there are no enough data in the buffers read returns nil slice and nil error
func (s *ZeroCopyStream) read(samples int) ([]int16, error) {
	return s.processAndRead(samples)
}

// readTo reads samples into the provided `to` slice. It returns the buffer filled with audio data.
func (s *ZeroCopyStream) readTo(to []int16) ([]int16, error) {
	data, err := s.read(cap(to))
	if err != nil {
		return to[:0], err
	}
	to = to[:len(data)]
	copy(to, data)

	return to, nil
}
