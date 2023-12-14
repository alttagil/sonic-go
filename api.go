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

// ChangeSpeed modifies the speed, pitch, rate, and volume of the provided int16 samples.
// It returns the modified int16 samples and any encountered error.
func ChangeSpeed(sampleRate, numChannels int, speed, pitch, rate, volume float64, samples []int16) ([]int16, error) {
	stream := NewSonicStream(sampleRate, numChannels)
	stream.SetSpeed(speed)
	stream.SetPitch(pitch)
	stream.SetRate(rate)
	stream.SetVolume(volume)
	if err := stream.AddSamples(samples); err != nil {
		return samples, err
	}
	if err := stream.Flush(); err != nil {
		return samples, err
	}
	out, err := stream.ReadAll()
	if err != nil {
		return samples, err
	}

	if cap(samples) < len(out) {
		samples = make([]int16, len(out))
	} else {
		samples = samples[:len(out)-1]
	}

	n := copy(samples, out)
	return samples[:n-1], nil
}

// ChangeFloatSpeed modifies the speed, pitch, rate, and volume of the provided float64 samples.
// It returns the modified float64 samples and any encountered error.
func ChangeFloatSpeed(sampleRate, numChannels int, speed, pitch, rate, volume float64, samples []float64) ([]float64, error) {
	stream := NewSonicStream(sampleRate, numChannels)
	stream.SetSpeed(speed)
	stream.SetPitch(pitch)
	stream.SetRate(rate)
	stream.SetVolume(volume)
	if err := stream.AddFloatSamples(samples); err != nil {
		return samples, err
	}
	if err := stream.Flush(); err != nil {
		return samples, err
	}
	out, err := stream.ReadAll()
	if err != nil {
		return samples, err
	}

	if cap(samples) < len(out) {
		samples = make([]float64, len(out))
	} else {
		samples = samples[:len(out)-1]
	}

	for i := 0; i <= cap(samples) && i <= len(out); i++ {
		samples[i] = float64(out[i]) / 32767.0
	}

	return samples, nil
}

// ChangeByteSpeed modifies the speed, pitch, rate, and volume of the provided uint8 samples.
// It returns the modified uint8 samples and any encountered error.
func ChangeByteSpeed(sampleRate, numChannels int, speed, pitch, rate, volume float64, samples []uint8) ([]uint8, error) {
	stream := NewSonicStream(sampleRate, numChannels)
	stream.SetSpeed(speed)
	stream.SetPitch(pitch)
	stream.SetRate(rate)
	stream.SetVolume(volume)
	if err := stream.AddByteSamples(samples); err != nil {
		return samples, err
	}
	if err := stream.Flush(); err != nil {
		return samples, err
	}

	out, err := stream.ReadAll()
	if err != nil {
		return samples, err
	}

	if cap(samples) < len(out) {
		samples = make([]uint8, len(out))
	} else {
		samples = samples[:len(out)-1]
	}

	for i := 0; i <= cap(samples) && i <= len(out); i++ {
		samples[i] = uint8(out[i]>>8) + 128
	}

	return samples, nil
}

// Write writes int16 samples to the Stream and process data.
// It returns any encountered error during the process.
func (stream *Stream) Write(samples []int16) error {
	if err := stream.AddSamples(samples); err != nil {
		return err
	}
	return stream.processStreamInput()
}

// WriteFloats writes float64 samples to the Stream and process data.
// It returns any encountered error during the process.
func (stream *Stream) WriteFloats(samples []float64) error {
	if err := stream.AddFloatSamples(samples); err != nil {
		return err
	}
	return stream.processStreamInput()
}

// WriteBytes writes uint8 samples to the Stream and process data.
// It returns any encountered error during the process.
func (stream *Stream) WriteBytes(samples []uint8) error {
	if err := stream.AddByteSamples(samples); err != nil {
		return err
	}
	return stream.processStreamInput()
}

// Read reads a slice wih a len n from the outputBuffer
func (stream *Stream) Read(n int) ([]int16, error) {
	return stream.outputBuffer.ReadSlice(n)
}

// ReadAll flushes and returns slice with all the data in the outputBuffer
func (stream *Stream) ReadAll() ([]int16, error) {
	return stream.outputBuffer.Flush()
}

// ReadTo reads data from the outputBuffer to a slice
func (stream *Stream) ReadTo(s []int16) ([]int16, error) {
	n := cap(s) / stream.numChannels
	if n == 0 {
		return s[:0], nil
	}

	data, err := stream.outputBuffer.ReadSlice(n)
	if err != nil {
		return s[:0], err
	}
	s = s[:len(data)]
	copy(s, data)

	return s, nil
}

// NumInputSamples returns number of samples in input buffer
func (stream *Stream) NumInputSamples() int {
	return stream.inputBuffer.Len()
}

// NumOutputSamples returns number of samples in output buffer
func (stream *Stream) NumOutputSamples() int {
	return stream.outputBuffer.Len()
}
