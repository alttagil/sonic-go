package sonic

type Stream struct {
	*Sonic
}

// NewSonicStream creates a new sonic NonCopy Sonic.
func NewSonicStream(sampleRate, numChannels int) *Stream {
	return &Stream{NewSonic(sampleRate, numChannels)}
}

// Write writes int16 samples to the Sonic and process data.
// It returns any encountered error during the process.
func (s *Stream) Write(samples []int16) error {
	if err := s.AddSamples(samples); err != nil {
		return err
	}
	return s.processStreamInput()
}

// WriteFloats writes float64 samples to the Sonic and process data.
// It returns any encountered error during the process.
func (s *Stream) WriteFloats(samples []float64) error {
	if err := s.AddFloatSamples(samples); err != nil {
		return err
	}
	return s.processStreamInput()
}

// WriteBytes writes uint8 samples to the Sonic and process data.
// It returns any encountered error during the process.
func (s *Stream) WriteBytes(samples []uint8) error {
	if err := s.AddByteSamples(samples); err != nil {
		return err
	}
	return s.processStreamInput()
}

// Read reads a slice wih a len n from the outputBuffer
func (s *Stream) Read(n int) ([]int16, error) {
	return s.outputBuffer.ReadSlice(n)
}

// ReadAll flushes and returns slice with all the data in the outputBuffer
func (s *Stream) ReadAll() ([]int16, error) {
	return s.outputBuffer.Flush()
}

// ReadTo reads data from the outputBuffer to a slice
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

// NumInputSamples returns number of samples in input buffer
func (s *Stream) NumInputSamples() int {
	return s.inputBuffer.Len()
}

// NumOutputSamples returns number of samples in output buffer
func (s *Stream) NumOutputSamples() int {
	return s.outputBuffer.Len()
}

// AddSamples adds int16 samples to the inputBuffer
func (s *Stream) AddSamples(samples []int16) error {
	if err := s.inputBuffer.AddSamples(samples); err != nil {
		return err
	}
	s.inputPlaytime = float64(s.inputSamplesLen()) * s.samplePeriod / (s.speed / s.pitch)
	return nil
}

// AddSamples coverts float64 samples to the int16 samples and add them to the inputBuffer
func (s *Stream) AddFloatSamples(samples []float64) error {
	if err := s.inputBuffer.AddFloatSamples(samples); err != nil {
		return err
	}
	s.inputPlaytime = float64(s.inputSamplesLen()) * s.samplePeriod / (s.speed / s.pitch)
	return nil
}

// AddSamples coverts uint8 samples to the int16 samples and add them to the inputBuffer
func (s *Stream) AddByteSamples(samples []uint8) error {
	if err := s.inputBuffer.AddByteSamples(samples); err != nil {
		return err
	}
	s.inputPlaytime = float64(s.inputSamplesLen()) * s.samplePeriod / (s.speed / s.pitch)
	return nil
}
