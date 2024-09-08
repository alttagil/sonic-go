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
	"errors"
	"fmt"
)

var ErrChannels = errors.New("incompatible number of elements for the specified number of channels")

// SampleBuffer represents a buffer for audio samples. Built upon a sonic.Buffer implementation
type SampleBuffer struct {
	*Buffer[int16]         // Embedding Buffer[int16] to inherit its methods and fields
	ch             int     // Number of channels
	empty          []int16 // Slice of empty samples for efficient use in WriteEmpty
}

// NewSampleBuffer creates a new SampleBuffer with the specified number of channels and capacity.
//
// Parameters:
//
//	ch:       int - The number of channels.
//	capacity: int - The initial capacity of the SampleBuffer.
//
// Returns:
//
//	*SampleBuffer: The newly created SampleBuffer.
func NewSampleBuffer(ch, capacity int) *SampleBuffer {
	return &SampleBuffer{
		Buffer: NewBuffer[int16](capacity * ch),
		ch:     ch,
		empty:  make([]int16, 4096),
	}
}

// RawSlice (EXPERIMENTAL) returns slice from a buffer without counter changes
func (b *SampleBuffer) RawSlice(n int) []int16 {
	return b.Buffer.RawSlice(n * b.ch)
}

// RawLenAdd (EXPERIMENTAL) changes internal counters as if n samples were added to a buffer
func (b *SampleBuffer) RawLenAdd(n int) error {
	return b.Buffer.RawLenAdd(n * b.ch)
}

// Channels returns the number of channels in the SampleBuffer.
func (b *SampleBuffer) Channels() int {
	return b.ch
}

// Len returns the number of samples in the buffer.
func (b *SampleBuffer) Len() int {
	return b.Buffer.Len() / b.ch
}

// Available returns the number of available samples in the buffer.
func (b *SampleBuffer) Available() int {
	return b.Buffer.Available() / b.ch
}

// Truncate truncates the buffer to the specified number of samples.
//
// Parameters:
//
//	n: The number of samples to retain in the buffer after truncation.
func (b *SampleBuffer) Truncate(n int) {
	b.Buffer.Truncate(n * b.ch)
}

// DropSlice drops the specified number of samples from the buffer.
//
// Parameters:
//
//	n:   The number of samples to drop from the buffer.
func (b *SampleBuffer) DropSlice(n int) error {
	return b.Buffer.DropSlice(n * b.ch)
}

// SetChannel sets the value of a sample in the specified channel at the given position.
//
// Parameters:
//
//	at:   The position in the SampleBuffer for which the sample value is set.
//	ch:   The channel for which the sample value is set.
//	val:  The new value to set for the sample.
func (b *SampleBuffer) SetChannel(at, ch int, val int16) {
	b.Buffer.WriteAt(at*b.ch+ch, val)
}

// GetChannel returns the value of a sample in the specified channel at the given position.
//
// Parameters:
//
//	at: The position in the SampleBuffer for which the sample value is retrieved.
//	ch: The channel for which the sample value is retrieved.
func (b *SampleBuffer) GetChannel(at, ch int) (int16, error) {
	return b.Buffer.At(at*b.ch + ch)
}

// Scale scales the samples starting from the specified position with the given factor.
//
// Parameters:
//
//	at: The position in the SampleBuffer from which scaling starts.
//	factor: The scaling factor to be applied to each sample.
func (b *SampleBuffer) Scale(at, factor int) error {
	slice, err := b.ReadSliceAt(at)
	if err == nil {
		for i := range slice {
			slice[i] = scaleInt16(factor, slice[i])
		}
		return b.WriteSlice(slice)
	}
	return err
}

// WriteSlice writes a slice of samples to the SampleBuffer.
//
// Parameters:
//
//	s: The slice of samples to be written.
func (b *SampleBuffer) WriteSlice(s []int16) error {
	if len(s)%b.ch != 0 {
		return ErrChannels
	}
	return b.Buffer.WriteSlice(s)
}

// Flush reads and returns a slice containing all samples from the SampleBuffer.
func (b *SampleBuffer) Flush() ([]int16, error) {
	return b.Buffer.ReadSlice(b.Buffer.Len())
}

// WriteEmpty writes empty audio samples to the SampleBuffer, increasing its length.
//
// Parameters:
//
//	n - The number of empty samples to write to the SampleBuffer.
//
// Returns:
//
//	The length of the SampleBuffer before writing the empty samples.
func (b *SampleBuffer) WriteEmpty(n int) (int, error) {
	cur := b.Len()

	num := n * b.ch
	if len(b.empty) < num {
		b.empty = make([]int16, num+1024)
	}

	err := b.Buffer.WriteSlice(b.empty[:num])
	return cur, err
}

// GetSlice gets and returns a slice of audio samples from the current SampleBuffer
// without removing them from the underlying Buffer. The length of the resulting slice
// is determined by the specified number of samples 'n' and the number of channels in the SampleBuffer.
//
// Parameters:
//
//	n - The number of samples to retrieve from the SampleBuffer.
func (b *SampleBuffer) GetSlice(n int) ([]int16, error) {
	return b.Buffer.GetSlice(n * b.ch)
}

// ReadSlice reads and returns a slice of audio samples from the current SampleBuffer.
// The length of the resulting slice is determined by the specified number of samples 'n'
// and the number of channels in the SampleBuffer.
//
// Parameters:
//
//	n - The number of samples to read from the SampleBuffer.
func (b *SampleBuffer) ReadSlice(n int) ([]int16, error) {
	return b.Buffer.ReadSlice(n * b.ch)
}

// ReadSliceAt reads and returns a slice of audio samples from the current SampleBuffer,
// starting from the specified position 'n' measured in samples.
// The number of channels in the resulting slice is determined by the number of channels in the SampleBuffer.
//
// Parameters:
//
//	n - The starting position in samples from which to read the slice.
func (b *SampleBuffer) ReadSliceAt(n int) ([]int16, error) {
	return b.Buffer.ReadSliceAt(n * b.ch)
}

// CopyTo copies audio samples from the current SampleBuffer to another SampleBuffer.
// It ensures that the destination SampleBuffer has the same number of channels.
// The parameter 'n' specifies the number of samples to be copied from the current buffer to the destination buffer.
// If the current SampleBuffer is empty or 'n' is zero, this operation has no effect.
//
// Parameters:
//
//	dest - The destination SampleBuffer to which the samples will be copied.
//	n - The number of samples to copy.
func (b *SampleBuffer) CopyTo(dest *SampleBuffer, n int) error {
	if b.ch != dest.Channels() {
		return fmt.Errorf("wrong number of channels %d", dest.Channels())
	}
	return b.Buffer.CopyTo(dest.Buffer, n*b.ch)
}

// MoveTo moves audio samples from the current SampleBuffer to another SampleBuffer.
// It ensures that the destination SampleBuffer has the same number of channels.
// The parameter 'n' specifies the number of samples to be moved from the current buffer to the destination buffer.
// If the current SampleBuffer is empty or 'n' is zero, this operation has no effect.
//
// Parameters:
//
//	dest - The destination SampleBuffer to which the samples will be moved.
//	n - The number of samples to move.
func (b *SampleBuffer) MoveTo(dest *SampleBuffer, n int) error {
	if b.ch != dest.Channels() {
		return fmt.Errorf("wrong number of channels %d", dest.Channels())
	}
	return b.Buffer.MoveTo(dest.Buffer, n*b.ch)
}

// MoveAllTo moves all audio samples from the current SampleBuffer to another SampleBuffer.
// It ensures that the destination SampleBuffer has the same number of channels.
// If the current SampleBuffer is empty, this operation has no effect.
//
// Parameters:
//
//	dest - The destination SampleBuffer to which the samples will be moved.
func (b *SampleBuffer) MoveAllTo(dest *SampleBuffer) error {
	if b.ch != dest.Channels() {
		return fmt.Errorf("wrong number of channels %d", dest.Channels())
	}
	return b.Buffer.MoveAllTo(dest.Buffer)
}

// AddFloatSamples appends the specified float64 samples to the SampleBuffer.
// Each sample is scaled to the range of int16 and added to the buffer.
// The number of elements in the input slice must be a multiple of the buffer's channel count.
func (b *SampleBuffer) AddFloatSamples(s []float64) error {
	if len(s)%b.ch != 0 {
		return ErrChannels
	}
	b.Buffer.Grow(len(s) / b.ch)
	for i := range s {
		if err := b.Buffer.Write(int16(s[i] * 32767.0)); err != nil {
			return err
		}
	}
	return nil
}

// AddByteSamples appends the specified uint8 samples to the SampleBuffer.
// Each uint8 sample is converted to an int16 value and shifted to the range of int16.
// The number of elements in the input slice must be a multiple of the buffer's channel count.
func (b *SampleBuffer) AddByteSamples(s []uint8) error {
	if len(s)%b.ch != 0 {
		return ErrChannels
	}
	b.Buffer.Grow(len(s) / b.ch)
	for i := range s {
		if err := b.Buffer.Write((int16(s[i]) - 128) << 8); err != nil {
			return err
		}
	}
	return nil
}

// AddSamples appends the specified int16 samples to the SampleBuffer.
// The samples are directly written to the buffer.
func (b *SampleBuffer) AddSamples(s []int16) error {
	return b.WriteSlice(s)
}

// scaleInt16 scales the given int16 sample by the specified volume factor.
// The result is clamped to the valid int16 range.
// This function is used internally for scaling audio samples in the SampleBuffer.
func scaleInt16(volume int, sample int16) int16 {
	val := (volume * int(sample)) >> 8
	if val > ShrtMax {
		val = ShrtMax
	} else if val < ShrtMin {
		val = ShrtMin
	}
	return int16(val)
}
