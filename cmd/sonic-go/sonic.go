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

package main

import (
	"flag"
	"github.com/alttagil/sonic-go"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"log"
	"os"
	"time"
)

const BufLen = 4096

var IntBuf = make([]int, BufLen)

func main() {
	pitch := flag.Float64("p", 1.0, "Set pitch scaling factor.  1.3 means 30%% higher.")
	rate := flag.Float64("r", 1.0, "Set playback rate.  2.0 means 2X faster, and 2X pitch.")
	speed := flag.Float64("s", 1.0, "Set speed up factor.  2.0 means 2X faster.")
	volume := flag.Float64("v", 1.0, "Set volume scale factor.  2.0 means 2X louder.")
	in := flag.String("i", "", "Input WAV filename")
	out := flag.String("o", "out.wav", "Output WAV filename")

	flag.Parse()

	f, err := os.Open(*in)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	decoder := wav.NewDecoder(f)
	decoder.ReadInfo()
	format := decoder.Format()

	stream := sonic.NewSonicStream(int(format.SampleRate), int(format.NumChannels))
	stream.SetPitch(*pitch)
	stream.SetSpeed(*speed)
	stream.SetRate(*rate)
	stream.SetVolume(*volume)

	of, err := os.Create(*out)
	if err != nil {
		log.Fatalln(err)
	}
	defer of.Close()

	enc := wav.NewEncoder(of, format.SampleRate, 16, format.NumChannels, 1)
	defer enc.Close()

	intBufSamplesNum := len(IntBuf) / format.NumChannels

	s := make([]int16, 0, BufLen)
	var elapsedTime time.Duration

	buf := &audio.IntBuffer{Data: IntBuf}
	for {
		samples, _ := decoder.PCMBuffer(buf)
		if samples == 0 {
			break
		}

		if buf.SourceBitDepth > 16 {
			log.Fatalln("Not supported bit depth", buf.SourceBitDepth)
		}

		s = s[:0]
		for i := 0; i < samples; i++ {
			s = append(s, int16(buf.Data[i]))
		}

		startTime := time.Now()
		if err := stream.Write(s); err != nil {
			log.Fatalln(err)
		}
		elapsedTime += time.Since(startTime)

		writeSamples(stream, enc, format, intBufSamplesNum)
	}

	startTime := time.Now()
	if err := stream.Flush(); err != nil {
		log.Fatalln(err)
	}
	elapsedTime += time.Since(startTime)

	writeSamples(stream, enc, format, intBufSamplesNum)

	log.Println("Processed in", elapsedTime)
}

func writeSamples(stream *sonic.Stream, enc *wav.Encoder, format *audio.Format, n int) {
	for {
		outs, err := stream.Read(n)
		if err != nil {
			break
		}

		IntBuf = IntBuf[:0]
		for i := range outs {
			IntBuf = append(IntBuf, int(outs[i]))
		}

		if err := enc.Write(&audio.IntBuffer{
			Format:         format,
			SourceBitDepth: 16,
			Data:           IntBuf,
		}); err != nil {
			log.Fatalln(err)
		}
	}
}
