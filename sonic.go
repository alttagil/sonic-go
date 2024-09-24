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

/*
#include <stdint.h>
#include <stdlib.h>
#include <math.h>

struct Result {
    int bestPeriod;
    int minDiff;
    int maxDiff;
};

struct Result findPitchPeriod(int16_t* samples, int minP, int maxP) {
    struct Result result;

    int period;
    int bestPeriod = 0;
    int worstPeriod = 255;
    unsigned long diff, minDiff = 1, maxDiff = 0;

    for (int period = minP; period <= maxP; period++) {
        int diff = 0;
        for (int i = 0; i < period; i++) {
            diff += abs(samples[i] - samples[i + period]);
        }

        if (bestPeriod == 0 || diff * bestPeriod < minDiff * period) {
            minDiff = diff;
            bestPeriod = period;
        }

        if (diff * worstPeriod > maxDiff * period) {
            maxDiff = diff;
            worstPeriod = period;
        }
    }

    result.minDiff = minDiff / bestPeriod;
    result.maxDiff = maxDiff / worstPeriod;
    result.bestPeriod = bestPeriod;

    return result;
}
*/
import "C"

import (
	"math"
)

const (
	// MinPitch specifies the range of voice pitches we try to match.
	// Note that if we go lower than 65, we could overflow in findPitchInRange
	MinPitch = 65

	// MaxPitch specifies the upper limit of voice pitches we try to match.
	MaxPitch = 400

	// AmdfFreq are used to down-sample some inputs to improve speed
	AmdfFreq = 4000

	// SincFilterPoints is a number of points to use in the sinc FIR filter for resampling.
	SincFilterPoints = 12
	SincTableSize    = 601

	// ShrtMax represents the maximum positive value for a signed 16-bit integer.
	ShrtMax = 32767
	// ShrtMin represents the minimum negative value for a signed 16-bit integer.
	ShrtMin = -32768
)

/*
	The following code was used to generate the following sinc lookup table:

	package main

	import (
		"fmt"
		"math"
	)

	func findHannWeight(N int, x float64) float64 {
		return 0.5 * (1.0 - math.Cos(2*math.Pi*x/float64(N)))
	}

	func findSincCoefficient(N int, x float64) float64 {
		hannWindowWeight := findHannWeight(N, x)
		var sincWeight float64

		x -= float64(N) / 2.0
		if math.Abs(x) > 1e-9 {
			sincWeight = math.Sin(math.Pi*x) / (math.Pi * x)
		} else {
			sincWeight = 1.0
		}
		return hannWindowWeight * sincWeight
	}

	func main() {
		var x float64
		N := 12

		for i := 0; x <= float64(N); x += 0.02 {
			fmt.Printf("%d, ", int(math.MaxInt16*findSincCoefficient(N, x)))
			i++
			if i%10 == 0 {
				fmt.Printf("\n")
			}

		}
	}
*/

var SincTable = [SincTableSize]int{
	0, 0, 0, 0, 0, 0, 0, -1, -1, -2, -2, -3, -4, -6, -7, -9, -10, -12, -14, -17,
	-19, -21, -24, -26, -29, -32, -34, -37, -40, -42, -44, -47, -48, -50, -51, -52, -53, -53, -53, -52,
	-50, -48, -46, -43, -39, -34, -29, -22, -16, -8, 0, 9, 19, 29, 41, 53, 65, 79, 92, 107,
	121, 137, 152, 168, 184, 200, 215, 231, 247, 262, 276, 291, 304, 317, 328, 339, 348, 357, 363, 369,
	372, 374, 375, 373, 369, 363, 355, 345, 332, 318, 300, 281, 259, 234, 208, 178, 147, 113, 77, 39,
	0, -41, -85, -130, -177, -225, -274, -324, -375, -426, -478, -530, -581, -632, -682, -731, -779, -825, -870, -912,
	-951, -989, -1023, -1053, -1080, -1104, -1123, -1138, -1149, -1154, -1155, -1151, -1141, -1125, -1105, -1078, -1046, -1007, -963, -913,
	-857, -796, -728, -655, -576, -492, -403, -309, -210, -107, 0, 111, 225, 342, 462, 584, 708, 833, 958, 1084,
	1209, 1333, 1455, 1575, 1693, 1807, 1916, 2022, 2122, 2216, 2304, 2384, 2457, 2522, 2579, 2625, 2663, 2689, 2706, 2711,
	2705, 2687, 2657, 2614, 2559, 2491, 2411, 2317, 2211, 2092, 1960, 1815, 1658, 1489, 1308, 1115, 912, 698, 474, 241,
	0, -249, -506, -769, -1037, -1310, -1586, -1864, -2144, -2424, -2703, -2980, -3254, -3523, -3787, -4043, -4291, -4529, -4757, -4972,
	-5174, -5360, -5531, -5685, -5819, -5935, -6029, -6101, -6150, -6175, -6175, -6149, -6096, -6015, -5905, -5767, -5599, -5401, -5172, -4912,
	-4621, -4298, -3944, -3558, -3141, -2693, -2214, -1705, -1166, -597, 0, 625, 1277, 1955, 2658, 3386, 4135, 4906, 5697, 6506,
	7332, 8173, 9027, 9893, 10769, 11654, 12544, 13439, 14335, 15232, 16128, 17019, 17904, 18782, 19649, 20504, 21345, 22170, 22977, 23763,
	24527, 25268, 25982, 26669, 27327, 27953, 28547, 29107, 29632, 30119, 30569, 30979, 31349, 31678, 31964, 32208, 32408, 32565, 32677, 32744,
	32767, 32744, 32677, 32565, 32408, 32208, 31964, 31678, 31349, 30979, 30569, 30119, 29632, 29107, 28547, 27953, 27327, 26669, 25982, 25268,
	24527, 23763, 22977, 22170, 21345, 20504, 19649, 18782, 17904, 17019, 16128, 15232, 14335, 13439, 12544, 11654, 10769, 9893, 9027, 8173,
	7332, 6506, 5697, 4906, 4135, 3386, 2658, 1955, 1277, 625, 0, -597, -1166, -1705, -2214, -2693, -3141, -3558, -3944, -4298,
	-4621, -4912, -5172, -5401, -5599, -5767, -5905, -6015, -6096, -6149, -6175, -6175, -6150, -6101, -6029, -5935, -5819, -5685, -5531, -5360,
	-5174, -4972, -4757, -4529, -4291, -4043, -3787, -3523, -3254, -2980, -2703, -2424, -2144, -1864, -1586, -1310, -1037, -769, -506, -249,
	0, 241, 474, 698, 912, 1115, 1308, 1489, 1658, 1815, 1960, 2092, 2211, 2317, 2411, 2491, 2559, 2614, 2657, 2687,
	2705, 2711, 2706, 2689, 2663, 2625, 2579, 2522, 2457, 2384, 2304, 2216, 2122, 2022, 1916, 1807, 1693, 1575, 1455, 1333,
	1209, 1084, 958, 833, 708, 584, 462, 342, 225, 111, 0, -107, -210, -309, -403, -492, -576, -655, -728, -796,
	-857, -913, -963, -1007, -1046, -1078, -1105, -1125, -1141, -1151, -1155, -1154, -1149, -1138, -1123, -1104, -1080, -1053, -1023, -989,
	-951, -912, -870, -825, -779, -731, -682, -632, -581, -530, -478, -426, -375, -324, -274, -225, -177, -130, -85, -41,
	0, 39, 77, 113, 147, 178, 208, 234, 259, 281, 300, 318, 332, 345, 355, 363, 369, 373, 375, 374,
	372, 369, 363, 357, 348, 339, 328, 317, 304, 291, 276, 262, 247, 231, 215, 200, 184, 168, 152, 137,
	121, 107, 92, 79, 65, 53, 41, 29, 19, 9, 0, -8, -16, -22, -29, -34, -39, -43, -46, -48,
	-50, -52, -53, -53, -53, -52, -51, -50, -48, -47, -44, -42, -40, -37, -34, -32, -29, -26, -24, -21,
	-19, -17, -14, -12, -10, -9, -7, -6, -4, -3, -2, -2, -1, -1, 0, 0, 0, 0, 0, 0, 0,
}

// Sonic represents the internal structure of a Sonic stream.
type Sonic struct {
	// inputBuffer holds the input samples.
	inputBuffer *SampleBuffer

	// outputBuffer holds the output samples.
	outputBuffer *SampleBuffer

	// pitchBuffer is used for pitch adjustment.
	pitchBuffer *SampleBuffer

	// downSampleBuffer is used for down-sampling.
	downSampleBuffer *SampleBuffer

	// speed is the playback speed factor.
	speed float64

	// volume is the volume adjustment factor.
	volume float64

	// pitch is the pitch adjustment factor.
	pitch float64

	// rate is the playback rate adjustment factor.
	rate float64

	// erate is a calculated effective rate
	erate float64

	// samplePeriod is the duration of each output sample, calculated as 1.0 / sampleRate.
	// It is used in accumulating inputPlaytime.
	samplePeriod float64

	// inputPlaytime represents how long the entire input buffer is expected to take to play.
	inputPlaytime float64

	// timeError keeps track of the error in playtime created when playing < 2.0X speed.
	// In cases where a whole pitch period is inserted or deleted, this can cause the output
	// generated from the input to be off in playtime by up to a pitch period. timeError replaces
	// PICOLA's concept of the number of samples to play unmodified after a pitch period insertion
	// or deletion. If speeding up, and the error is >= 0.0, then a pitch period is removed, and
	// samples are played unmodified until timeError is >= 0 again. If slowing down, and the error
	// is <= 0.0, then a pitch period is added, and samples are played unmodified until timeError
	// is <= 0 again.
	timeError float64

	// oldRatePosition is the previous position in the rate buffer.
	oldRatePosition int

	// newRatePosition is the current position in the rate buffer.
	newRatePosition int

	// quality indicates the quality mode of the Sonic stream.
	quality bool

	// numChannels is the number of audio channels.
	numChannels int

	// minPeriod is the minimum pitch period.
	minPeriod int

	// maxPeriod is the maximum pitch period.
	maxPeriod int

	// maxRequired is the maximum required size of the pitch buffer.
	maxRequired int

	// sampleRate is the audio sample rate.
	sampleRate int

	// prevPeriod is the previous pitch period.
	prevPeriod int

	// prevMinDiff is the previous minimum difference.
	prevMinDiff int

	// useSinOverlap - set UseSinOverlap to true to use sin-wav based overlap add which in theory can improve
	// sound quality slightly, at the expense of lots of floating point math.
	useSinOverlap bool
}

// NewSonicStream creates a new sonic Sonic.
func NewSonic(sampleRate, numChannels int) *Sonic {
	minPeriod := sampleRate / MaxPitch
	maxPeriod := sampleRate / MinPitch
	maxRequired := 2 * maxPeriod

	bufferSize := (maxRequired + (maxRequired >> 2)) * numChannels

	skip := 1
	if sampleRate > AmdfFreq {
		skip = sampleRate / AmdfFreq
	}
	downSamplerBufferSize := (maxRequired + skip - 1) / skip

	stream := &Sonic{
		sampleRate:       sampleRate,
		numChannels:      numChannels,
		minPeriod:        minPeriod,
		maxPeriod:        maxPeriod,
		maxRequired:      maxRequired,
		inputBuffer:      NewSampleBuffer(numChannels, bufferSize),
		outputBuffer:     NewSampleBuffer(numChannels, bufferSize),
		pitchBuffer:      NewSampleBuffer(numChannels, bufferSize),
		downSampleBuffer: NewSampleBuffer(1, downSamplerBufferSize),
		samplePeriod:     1.0 / float64(sampleRate),

		speed:   1.0,
		pitch:   1.0,
		volume:  1.0,
		rate:    1.0,
		erate:   1.0,
		quality: false,

		prevPeriod:      0,
		oldRatePosition: 0,
		newRatePosition: 0,
	}
	return stream
}

// GetSpeed returns the speed of the stream.
func (s *Sonic) GetSpeed() float64 {
	return s.speed
}

// SetSpeed sets the speed of the stream.
func (s *Sonic) SetSpeed(speed float64) {
	s.speed = speed
	s.updateInputPlaytime()
}

// GetVolume returns the scaling factor of the stream.
func (s *Sonic) GetVolume() float64 {
	return s.volume
}

// SetVolume sets the volume
func (s *Sonic) SetVolume(volume float64) {
	s.volume = volume
}

// GetPitch returns the pitch of the stream.
func (s *Sonic) GetPitch() float64 {
	return s.pitch
}

// SetPitch sets the pitch of the stream.
func (s *Sonic) SetPitch(pitch float64) {
	s.pitch = pitch
	s.erate = s.rate * pitch
}

// GetRate returns the rate of the stream.
func (s *Sonic) GetRate() float64 {
	return s.rate
}

// GetSampleRate returns the sample rate of the stream.
func (s *Sonic) GetSampleRate() int {
	return s.sampleRate
}

// GetNumChannels returns the number of channels of the stream.
func (s *Sonic) GetNumChannels() int {
	return s.numChannels
}

// SetRate sets the playback rate of the stream. This scales pitch and speed at the same time.
func (s *Sonic) SetRate(rate float64) {
	s.rate = rate
	s.erate = rate * s.pitch
	s.oldRatePosition = 0
	s.newRatePosition = 0
}

// GetQuality returns the quality setting.
func (s *Sonic) GetQuality() bool {
	return s.quality
}

// SetQuality sets the "quality". Default false is virtually as good as true, but very much faster.
func (s *Sonic) SetQuality(quality bool) {
	s.quality = quality
}

// GetUseSinOverlap returns useSinOverlap value.
func (s *Sonic) GetUseSinOverlap() bool {
	return s.useSinOverlap
}

// SetUseSinOverlap sets the "useSinOverlap".
// Set UseSinOverlap to true to use sin-wav based overlap add which in theory can improve
// sound quality slightly, at the expense of lots of floating point math.
func (s *Sonic) SetUseSinOverlap(useSinOverlap bool) {
	s.useSinOverlap = useSinOverlap
}

// computeSkip computes the number of samples to skip to down-sample the input.
func (s *Sonic) computeSkip() int {
	skip := 1
	if s.sampleRate > AmdfFreq && !s.quality {
		skip = s.sampleRate / AmdfFreq
	}
	return skip
}

// inputSamplesLen is a helper function returning an inputBuffer len in samples
func (s *Sonic) inputSamplesLen() int {
	return s.inputBuffer.Len()
}

// moveInputToOutput moves all inputBuffer to outputBuffer
func (s *Sonic) moveInputToOutput() error {
	s.inputPlaytime = 0
	return s.inputBuffer.MoveAllTo(s.outputBuffer)
}

// moveUnmodifiedSamples moves samples should be left unmodified from inputBuffer to outputBuffer
func (s *Sonic) moveUnmodifiedSamples(speed float64) error {
	inputToCopyFloat := math.Round(1 - s.timeError*speed/(s.samplePeriod*(speed-1.0)))
	inputToCopy := int(inputToCopyFloat)

	var err error
	if inputToCopy > s.inputBuffer.Len() {
		inputToCopyFloat = float64(s.inputBuffer.Len())
		err = s.inputBuffer.MoveAllTo(s.outputBuffer)
	} else {
		err = s.inputBuffer.MoveTo(s.outputBuffer, inputToCopy)
	}

	s.timeError += inputToCopyFloat * s.samplePeriod * (speed - 1.0) / speed
	return err
}

// processStreamInput processes inputBuffer sampled changing its speed, rate, pitch, volume
func (s *Sonic) processStreamInput() error {
	InputLen := s.inputBuffer.Len()
	if InputLen == 0 {
		return nil
	}

	OutputLen := s.outputBuffer.Len()
	speed := float64(InputLen) * s.samplePeriod / s.inputPlaytime

	if speed > 1.00001 || speed < 0.99999 {
		if err := s.changeSpeed(speed); err != nil {
			return err
		}
	} else {
		if err := s.moveInputToOutput(); err != nil {
			return err
		}
	}

	if s.erate != 1.0 && OutputLen < s.outputBuffer.Len() {
		slice, err := s.outputBuffer.ReadSliceAt(OutputLen)
		if err != nil {
			return err
		}
		if err := s.adjustRate(s.erate, slice); err != nil {
			return err
		}
	}

	if s.volume != 1.0 && OutputLen < s.outputBuffer.Len() {
		fixedPointVolume := int(s.volume * 256.0)
		if err := s.outputBuffer.Scale(OutputLen, fixedPointVolume); err != nil {
			return err
		}
	}

	return nil
}

// adjustRate adjusts rate of the stream
func (s *Sonic) adjustRate(rate float64, slice []int16) error {
	newSampleRate := int(float64(s.sampleRate) / rate)
	oldSampleRate := s.sampleRate

	for newSampleRate > (1<<14) || oldSampleRate > (1<<14) {
		newSampleRate >>= 1
		oldSampleRate >>= 1
	}
	if err := s.pitchBuffer.WriteSlice(slice); err != nil {
		return err
	}

	// Leave at least SincFilterPoints pitch sample in the buffer
	blen := s.pitchBuffer.Len() - SincFilterPoints
	if blen < 1 {
		return nil
	}

	for i := 0; i < blen; i++ {
		for (s.oldRatePosition+1)*newSampleRate > s.newRatePosition*oldSampleRate {
			if err := s.interpolatePitch(i, oldSampleRate, newSampleRate); err != nil {
				return err
			}
		}
		s.oldRatePosition++
	}

	return s.pitchBuffer.DropSlice(blen)
}

// interpolatePitch interpolates along pitch period
func (s *Sonic) interpolatePitch(i, old, new int) error {
	cur, _ := s.outputBuffer.WriteEmpty(1)
	for c := 0; c < s.numChannels; c++ {
		value := s.interpolatePitchValue(i, c, old, new)
		s.outputBuffer.SetChannel(cur, c, value)
	}
	s.newRatePosition++
	return nil
}

// interpolatePitchValue interpolates the new output sample.
func (s *Sonic) interpolatePitchValue(n, c, old, new int) int16 {
	var overflowCount, total int
	position := s.newRatePosition * old
	leftPosition := s.oldRatePosition * new
	rightPosition := (s.oldRatePosition + 1) * new
	ratio := rightPosition - position - 1
	width := rightPosition - leftPosition

	for i := n; i < n+SincFilterPoints; i++ {
		weight := findSincCoefficient(i-n, ratio, width)
		chvalue, _ := s.pitchBuffer.GetChannel(i, c)
		value := int(chvalue) * weight
		oldSign := getSign(total)
		total += value
		if oldSign != getSign(total) && getSign(value) == oldSign {
			overflowCount += oldSign
		}
	}

	// It is better to clip than to wrap if there was an overflow.
	if overflowCount > 0 {
		return ShrtMax
	} else if overflowCount < 0 {
		return ShrtMin
	}

	return int16(total >> 16)
}

// findSincCoefficient approximates the sinc function times a Hann window from the sinc table.
func findSincCoefficient(i, ratio, width int) int {
	lobePoints := (SincTableSize - 1) / SincFilterPoints
	left := i*lobePoints + (ratio*lobePoints)/width
	position := i*lobePoints*width + ratio*lobePoints - left*width

	return ((SincTable[left]*(width-position) + SincTable[left+1]*position) << 1) / width
}

// getSign returns 1 if value >= 0, else -1.  This represents the sign of value.
func getSign(value int) int {
	if value >= 0 {
		return 1
	}
	return -1
}

// changeSpeed changes speed of the stream
func (s *Sonic) changeSpeed(speed float64) error {
	if s.inputSamplesLen() < s.maxRequired {
		return nil
	}

	playtime := s.inputPlaytime
	samplesNum := s.inputBuffer.Len()

	var period, newSamples int
	var err error
	for {
		if (speed > 1 && speed < 2 && s.timeError < 0) || (speed < 1 && speed > 0.5 && s.timeError > 0) {
			// Deal with the case where PICOLA is still copying input samples to
			// output unmodified,
			if err := s.moveUnmodifiedSamples(speed); err != nil {
				return err
			}
		} else {
			// We are in the remaining cases, either inserting/removing a pitch period
			// for speed < 2.0X, or a portion of one for speed >= 2.0X.
			period, err = s.findPitchPeriod(true)
			if err != nil {
				return err
			}

			if speed > 1 {
				newSamples, err = s.skipPitchPeriod(speed, period)
				if err != nil {
					return err
				}
				if speed < 2 {
					s.timeError += float64(newSamples)*s.samplePeriod - float64(period+newSamples)*playtime/float64(samplesNum)
				}
			} else {
				newSamples, err = s.insertPitchPeriod(speed, period)
				if err != nil {
					return err
				}
				if speed > 0.5 {
					s.timeError += float64(period+newSamples)*s.samplePeriod - float64(newSamples)*playtime/float64(samplesNum)
				}
			}
		}

		if newSamples == 0 {
			return nil
		}

		if s.inputSamplesLen() < s.maxRequired {
			break
		}
	}

	s.inputPlaytime = (playtime * float64(s.inputBuffer.Len())) / float64(samplesNum)
	return nil
}

// skipPitchPeriod skips over a pitch period.  Returns the number of output samples.
func (s *Sonic) skipPitchPeriod(speed float64, period int) (int, error) {
	var newSamples int
	if speed >= 2.0 {
		/* For speeds >= 2.0, we skip over a portion of each pitch period rather
		   than dropping whole pitch periods. */
		newSamples = int(math.Round(float64(period) / (speed - 1.0)))
	} else {
		newSamples = period
	}
	if err := s.overlapAdd(newSamples, period); err != nil {
		return 0, err
	}
	if err := s.inputBuffer.DropSlice(newSamples + period); err != nil {
		return 0, err
	}
	return newSamples, nil
}

// insertPitchPeriod inserts a pitch period, and determines how much input to copy directly.
func (s *Sonic) insertPitchPeriod(speed float64, period int) (int, error) {
	var newSamples int
	if speed <= 0.5 {
		newSamples = int(float64(period) * speed / (1.0 - speed))
	} else {
		newSamples = period
	}

	if err := s.inputBuffer.CopyTo(s.outputBuffer, period); err != nil {
		return 0, err
	}
	if err := s.overlapAdd(newSamples, period); err != nil {
		return 0, err
	}
	if err := s.inputBuffer.DropSlice(newSamples); err != nil {
		return 0, err
	}
	return newSamples, nil
}

// overlapAdd overlaps two sound segments, ramp the volume of one down, while ramping the
// other one from zero up, and add them, storing the result at the output.
func (s *Sonic) overlapAdd(numSamples int, period int) error {
	cur, _ := s.outputBuffer.WriteEmpty(numSamples)

	for i := 0; i < numSamples; i++ {
		for c := 0; c < s.numChannels; c++ {
			dv, _ := s.inputBuffer.GetChannel(i, c)
			uv, _ := s.inputBuffer.GetChannel(i+period, c)

			if s.useSinOverlap == true {
				ratio := math.Sin(float64(i) * math.Pi / (2 * float64(numSamples)))
				s.outputBuffer.SetChannel(cur+i, c, int16(float64(dv)*(1.0-ratio)+float64(uv)*ratio))
			} else {
				s.outputBuffer.SetChannel(cur+i, c, int16((int(dv)*(numSamples-i)+int(uv)*i)/numSamples))
			}
		}
	}
	return nil
}

func (s *Sonic) findPitchPeriod(preferNewPeriod bool) (int, error) {
	var period, minDiff, maxDiff, ret int

	minPeriod := s.minPeriod
	maxPeriod := s.maxPeriod
	skip := s.computeSkip()

	if s.numChannels == 1 && skip == 1 {
		period, minDiff, maxDiff = findPitchPeriodInRange(s.inputBuffer, minPeriod, maxPeriod)
	} else {
		if err := s.downSampleInput(skip); err != nil {
			return 0, err
		}
		period, minDiff, maxDiff = findPitchPeriodInRange(s.downSampleBuffer, minPeriod/skip, maxPeriod/skip)

		if skip != 1 {
			period *= skip
			minPeriod = period - (skip << 2)
			maxPeriod = period + (skip << 2)
			if minPeriod < s.minPeriod {
				minPeriod = s.minPeriod
			}
			if maxPeriod > s.maxPeriod {
				maxPeriod = s.maxPeriod
			}
			if s.numChannels == 1 {
				period, minDiff, maxDiff = findPitchPeriodInRange(s.inputBuffer, minPeriod, maxPeriod)
			} else {
				if err := s.downSampleInput(1); err != nil {
					return 0, err
				}
				period, minDiff, maxDiff = findPitchPeriodInRange(s.downSampleBuffer, minPeriod, maxPeriod)
			}
		}
	}

	if s.prevPeriodBetter(minDiff, maxDiff, preferNewPeriod) {
		ret = s.prevPeriod
	} else {
		ret = period
	}

	s.prevMinDiff = minDiff
	s.prevPeriod = period

	return ret, nil
}

// prevPeriodBetter detects At abrupt ends of voiced words, we can have pitch periods that are better
//
//	approximated by the previous pitch period estimate.  Try to detect this case.
func (s *Sonic) prevPeriodBetter(minDiff, maxDiff int, preferNewPeriod bool) bool {
	if minDiff == 0 || s.prevPeriod == 0 {
		return false
	}

	if preferNewPeriod {
		if maxDiff > minDiff*3 {
			/* Got a reasonable match this period */
			return false
		}
		if minDiff*2 <= s.prevMinDiff*3 {
			/* Mismatch is not that much greater this period */
			return false
		}
	} else {
		if minDiff <= s.prevMinDiff {
			return false
		}
	}
	return true
}

// downSampleInput down-samples inputBuffer:
// If skip is greater than one, average skip samples together and write them to the down-sample buffer.
// If numChannels is greater than one, mix the channels together as we down sample.
func (s *Sonic) downSampleInput(skip int) error {
	var n = s.maxRequired / skip
	s.downSampleBuffer.Truncate(0)

	buf, err := s.inputBuffer.GetSlice(s.maxRequired)
	if err != nil {
		return err
	}

	// _ = buf[s.maxRequired]

	skipCh := skip * s.numChannels

	ii := 0
	for i := 0; i < n; i++ {
		v := 0
		for j := 0; j < skipCh; j++ {
			v += int(buf[ii])
			ii++
		}
		v /= skipCh

		_ = s.downSampleBuffer.Write(int16(v))
	}
	return nil
}

// findPitchPeriodInRange finds the best frequency match in the range, and given a sample skip multiple.
// For now, just find the pitch of the first channel.
func findPitchPeriodInRange(b *SampleBuffer, minP, maxP int) (int, int, int) {
	samples, _ := b.GetSlice(2 * maxP)
	result := C.findPitchPeriod((*C.int16_t)(&samples[0]), C.int(minP), C.int(maxP))
	return int(result.bestPeriod), int(result.minDiff), int(result.maxDiff)
}

// Flush forces the sonic stream to generate output using whatever data it currently has.
// No extra delay will be added to the output, but flushing in the middle of words could introduce distortion.
func (s *Sonic) Flush() error {
	maxReq := s.maxRequired
	speed := s.speed / s.pitch
	expOutput := s.outputBuffer.Len() + int(math.Round((float64(s.inputBuffer.Len())/speed+float64(s.pitchBuffer.Len()))/s.erate+0.5))

	if err := s.AddEmptySamples(2 * maxReq * s.numChannels); err != nil {
		return err
	}

	if err := s.processStreamInput(); err != nil {
		return err
	}

	if s.outputBuffer.Len() > expOutput {
		s.outputBuffer.Truncate(expOutput)
	}

	s.inputPlaytime = 0
	s.timeError = 0

	return nil
}

// AddEmptySamples adds n empty samples to the inputBuffer
func (s *Sonic) AddEmptySamples(n int) error {
	if _, err := s.inputBuffer.WriteEmpty(n); err != nil {
		return err
	}
	s.updateInputPlaytime()
	return nil
}

// Reset instantly resets internal state and clears all buffers
func (s *Sonic) Reset() {
	s.prevPeriod = 0
	s.oldRatePosition = 0
	s.newRatePosition = 0
	s.timeError = 0
	s.inputPlaytime = 0

	s.inputBuffer.Reset()
	s.outputBuffer.Reset()
	s.downSampleBuffer.Reset()
	s.pitchBuffer.Reset()
}

func (s *Sonic) updateInputPlaytime() {
	s.inputPlaytime = float64(s.inputSamplesLen()) * s.samplePeriod * s.pitch / s.speed
}
