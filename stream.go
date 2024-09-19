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

type Stream struct {
	*Sonic
}

// NewSonicStream creates a new Stream instance, which wraps a Sonic instance.
// The stream is initialized with a specified sample rate and number of audio channels.
func NewSonicStream(sampleRate, numChannels int) *Stream {
	return &Stream{NewSonic(sampleRate, numChannels)}
}

// Write processes and writes a slice of `int16` audio samples into the Sonic input buffer.
// After adding the samples, it processes them. Returns an error if any issues occur during sample addition or processing.
func (s *Stream) Write(samples []int16) error {
	if err := s.AddSamples(samples); err != nil {
		return err
	}
	return s.processStreamInput()
}

// WriteFloats processes and writes a slice of `float64` audio samples into the Sonic input buffer.
// The samples are converted to `int16` before being added. After the conversion, it processes the data.
// Returns an error if any issues occur during sample conversion, addition, or processing.
func (s *Stream) WriteFloats(samples []float64) error {
	if err := s.AddFloatSamples(samples); err != nil {
		return err
	}
	return s.processStreamInput()
}

// WriteBytes processes and writes a slice of `uint8` audio samples into the Sonic input buffer.
// The samples are converted to `int16` before being added. After the conversion, it processes the data.
// Returns an error if any issues occur during sample conversion, addition, or processing.
func (s *Stream) WriteBytes(samples []uint8) error {
	if err := s.AddByteSamples(samples); err != nil {
		return err
	}
	return s.processStreamInput()
}

// Read retrieves and returns a slice of `int16` audio samples from the output buffer, with a length of `n` samples.
// Returns an error if any issues occur during the read operation.
func (s *Stream) Read(n int) ([]int16, error) {
	return s.outputBuffer.ReadSlice(n)
}

// ReadAll retrieves all the available audio samples from the output buffer and returns them as a slice of `int16`.
// This operation also flushes the buffer, removing the returned data from it.
// Returns an error if any issues occur during the read operation.
func (s *Stream) ReadAll() ([]int16, error) {
	return s.outputBuffer.Flush()
}

// ReadTo reads data from the output buffer and stores it into the provided `to` slice.
// The length of the data stored is determined by the capacity of `to`. The slice is resized to fit the read data.
// Returns an error if any issues occur during the read operation.
func (s *Stream) ReadTo(to []int16) ([]int16, error) {
	n := cap(to) / s.numChannels
	if n == 0 {
		return to[:0], nil
	}

	data, err := s.outputBuffer.ReadSlice(n)
	if err != nil {
		return to[:0], err
	}
	to = to[:len(data)-1]
	copy(to, data)

	return to, nil
}

// NumInputSamples returns the number of samples currently present in the input buffer.
// This provides information about how much unprocessed data is in the buffer.
func (s *Stream) NumInputSamples() int {
	return s.inputBuffer.Len()
}

// NumOutputSamples returns the number of samples currently present in the output buffer.
// This provides information about how much processed data is available for consumption.
func (s *Stream) NumOutputSamples() int {
	return s.outputBuffer.Len()
}

// AddSamples adds a slice of `int16` audio samples to the Sonic input buffer.
// It updates internal counters to reflect the new data and returns an error if any issues occur during addition.
func (s *Stream) AddSamples(samples []int16) error {
	if err := s.inputBuffer.AddSamples(samples); err != nil {
		return err
	}
	// Update internal playtime based on the number of samples and Sonic's speed/pitch settings
	s.inputPlaytime = float64(s.inputSamplesLen()) * s.samplePeriod / (s.speed / s.pitch)
	return nil
}

// AddFloatSamples converts a slice of `float64` samples to `int16` and adds them to the Sonic input buffer.
// It updates internal counters to reflect the new data and returns an error if any issues occur during conversion or addition.
func (s *Stream) AddFloatSamples(samples []float64) error {
	if err := s.inputBuffer.AddFloatSamples(samples); err != nil {
		return err
	}
	// Update internal playtime based on the number of samples and Sonic's speed/pitch settings
	s.inputPlaytime = float64(s.inputSamplesLen()) * s.samplePeriod / (s.speed / s.pitch)
	return nil
}

// AddByteSamples converts a slice of `uint8` samples to `int16` and adds them to the Sonic input buffer.
// It updates internal counters to reflect the new data and returns an error if any issues occur during conversion or addition.
func (s *Stream) AddByteSamples(samples []uint8) error {
	if err := s.inputBuffer.AddByteSamples(samples); err != nil {
		return err
	}
	// Update internal playtime based on the number of samples and Sonic's speed/pitch settings
	s.inputPlaytime = float64(s.inputSamplesLen()) * s.samplePeriod / (s.speed / s.pitch)
	return nil
}
