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
	s = s[:len(data)-1]
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

// Reset instantly resets internal state and clears all buffers
func (stream *Stream) Reset() {
	stream.prevPeriod = 0
	stream.oldRatePosition = 0
	stream.newRatePosition = 0
	stream.timeError = 0
	stream.inputPlaytime = 0

	stream.inputBuffer.Reset()
	stream.outputBuffer.Reset()
	stream.downSampleBuffer.Reset()
	stream.pitchBuffer.Reset()
}

// crossFade crossfades buf with tail. buf length must be greater or equal to tail length
func crossFade(buf, tail []int16) {
	l := len(tail)
	for i, decrV := range tail {
		incrV := buf[i]

		buf[i] = int16((int(decrV)*(l-i) + int(incrV)*i) / l)
	}
}

// StreamBorrowRawSlice (EXPERIMENTAL) borrows Raw slice from the sonic's input buffer for direct use, for example as audio decoding
// target memory.
// This slice must not be moved in case you want to return it back to the input buffer.
func (stream *Stream) StreamBorrowRawSlice(n int) []int16 {
	return stream.inputBuffer.RawSlice(n)
}

// StreamReturnRawSlice (EXPERIMENTAL) assumes that len(s) bytes are filled with audio data and adds up internal counters  
// to include audio data from s into internal buffer space.
// This function should be called right after corresponding StreamBorrowRawSlice. You can't borrow several slices and then 
// return it in bulk manner
func (stream *Stream) StreamReturnRawSlice(s []int16) error {
	n := len(s) / stream.numChannels
	if err := stream.inputBuffer.RawLenAdd(n); err != nil {
		return err
	}
	stream.inputPlaytime = float64(stream.inputSamplesLen()) * stream.samplePeriod / (stream.speed / stream.pitch)
	return nil
}

// StreamRead (EXPERIMENTAL) returns slice directly from input buffer if there is no any audio changes were expected (no speed, pitch or volume changes)
// Otherwise it works as usual - process ausio from input buffer to output and returns the slice from output buffer.
func (stream *Stream) StreamRead(s []int16) ([]int16, error) {
	n := cap(s) / stream.numChannels
	if n == 0 {
		return s[:0], nil
	}

	iLen := stream.inputBuffer.Len()
	oLen := stream.outputBuffer.Len()

	if iLen < n && oLen < n {
		return s[:0], nil
	}

	rate := stream.rate * stream.pitch
	speed := float64(iLen) * stream.samplePeriod / stream.inputPlaytime

	var data []int16
	var err error

	if speed > 0.99999 && speed < 1.00001 && rate == 1 && stream.volume == 1.0 {
		switch {
		case oLen == 0:
			data, err = stream.inputBuffer.ReadSlice(n)
			stream.inputPlaytime = float64(stream.inputSamplesLen()) * stream.samplePeriod / (stream.speed / stream.pitch)
		case oLen >= n:
			data, err = stream.outputBuffer.ReadSlice(n)
		default:
			if iLen < oLen {
				return s[:0], nil
			}

			data, err = stream.inputBuffer.ReadSlice(n)
			odata, _ := stream.outputBuffer.ReadSlice(n)
			crossFade(data, odata)

			stream.inputPlaytime = float64(stream.inputSamplesLen()) * stream.samplePeriod / (stream.speed / stream.pitch)
		}
	} else {
		if err := stream.processStreamInput(); err != nil {
			return s[:0], err
		} else {
			data, err = stream.outputBuffer.ReadSlice(n)
		}
	}

	if err != nil {
		return s[:0], err
	}
	s = s[:len(data)]
	copy(s, data)

	return s, nil
}

// StreamSamplesAvailable (EXPERIMENTAL)  returns count of samples available
func (stream *Stream) StreamSamplesAvailable() int {
	iLen := stream.inputBuffer.Len()
	oLen := stream.outputBuffer.Len()
	rate := stream.rate * stream.pitch
	speed := float64(iLen) * stream.samplePeriod / stream.inputPlaytime

	if speed > 0.99999 && speed < 1.00001 && rate == 1 && stream.volume == 1.0 {
		return max(iLen, oLen)
	}

	return oLen
}
