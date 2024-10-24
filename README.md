# Sonic Go Library

The Sonic Go Library is a Golang implementation of the Sonic algorithm, a simple yet effective method for speeding up or slowing down speech. Unlike earlier approaches to speech rate modification, Sonic is specifically optimized for significant speed changes, exceeding 2X. This library serves as a seamless integration into text-to-speech and voice streaming applications, offering the benefits of the original Sonic algorithm in a Go environment.

## Motivation

The primary motivation behind the Sonic Go Library is to provide Go developers with a straightforward and efficient tool for manipulating speech speed. The library is designed to be easily understandable, catering to Go developers and facilitating its use in various applications, including WebRTC SFUs and other real-time audio processing scenarios.

## Features

- Optimized algorithm for speeding up or slowing down speech.
- Integration-friendly Go library for effortless use in Go applications.
- Adaptation of the original Sonic library for enhanced compatibility with Go applications.
- Great for integration into text-to-speech applications and voice communication systems.

## Usage

The Sonic Go Library can be utilized in two main modes: streaming and batch processing.

### Streaming Mode

In streaming mode, you can initialize a Sonic stream, continuously feed it with input data, and read processed data in real-time. Here's a basic example:

```go
package main

import (
  "fmt"
  "github.com/alttagil/sonic-go"
)

func main() {
	stream := sonic.NewSonicStream(44100, 2) // Replace with your desired sample rate and number of channels
	stream.SetSpeed(1.5)
	
	// Simulates processing loop
	for {
		inputData, ok := GetInputDataSomewhere()
		if !ok {
			break;
		}

		// Write data to Sonic stream
		stream.Write(inputData)
		readAndProcess(stream)
	}
	
	if err := stream.Flush(); err != nil {
		log.Fatalln(err)
	}
	readAndProcess(stream)
}

func readAndProcess(stream *sonic.Stream) {
	for {
		processedData, err := stream.ReadTo(processedData)
		if err != nil || len(processedData) == 0 {
			break
		}

		// Simulated usage of processed data (replace with your actual usage)
		fmt.Println("Processed Data:", processedData)
	}
}
```

### Zero-Copy Streaming Mode
In zero-copy streaming mode, you process input data directly by borrowing a buffer, allowing for more efficient memory handling. Here's an example:

```go
package main

import (
  "fmt"
  "github.com/alttagil/sonic-go"
)

func main() {
    frameSize = 960
	
	stream := sonic.NewZeroCopyStream(44100, 2)  // Replace with your desired sample rate and number of channels
	stream.SetSpeed(1.5)
	
	// Simulates processing loop
	for {
      // Process input data in zero-copy mode and get the processed buffer
      tempAudioBuf, err := stream.Process(frameSize, prepareSamples)
      if err != nil {
        log.Fatalf("Error processing audio: %v", err)
      }
	  
      // Use the returned buffer for further processing (e.g., encoding)
      processSamples(tempAudioBuf)
	}
	
    if err := stream.Flush(); err != nil {
       log.Fatalln(err)
    }

    tempAudioBuf, err := stream.Process(frameSize, noop)
    if err != nil {
      log.Fatalf("Error processing audio: %v", err)
    }
    processSamples(tempAudioBuf)
}

func prepareSamples(buf []int16) error {
    inputData, err := GetInputDataSomewhere()
	if err != nil {
        return err
	}
	// Simulate decoding of input data into the borrowed buffer
    copy(buf, inputData)
    return nil
}

func noop(buf []int16) error {
	return nil
}

func processSamples(samples []int16) {
	if len(samples) == 0 {
		return
	}

    // Simulated usage of samples data (replace with your actual usage)
    fmt.Println("Processed Data:", samples)
}
```

### Batch Processing
Alternatively, you can use the library's function for batch processing:

```go
package main

import (
	"fmt"
	"github.com/alttagil/sonic-go"
)

func main() {
	sampleRate := 44100
	numChannels := 2
	speed := 1.0
	pitch := 1.0
	rate := 1.0
	volume := 1.0

	inputData := GetInputDataSomewhere() // Replace with your actual data

	// Process data using the ChangeSpeed function
	outputData, err := sonic.ChangeSpeed(sampleRate, numChannels, speed, pitch, rate, volume, inputData)
	if err != nil {
		fmt.Println("Error during processing:", err)
	}

	// Use the processed data as needed
	fmt.Println("Processed Data:", outputData)
}
```

### Parameters

In Sonic Go Library, the default configuration for a Sonic stream assumes no alterations to the sound stream, with speed, pitch, rate, and volume all set to 1.0. This signifies no change, and the library optimally handles this by directly copying input to output, minimizing CPU usage.

To modify the sonic characteristics, use the following functions:

```go
stream.SetSpeed(speed)
stream.SetPitch(pitch)
stream.SetRate(rate)
stream.SetVolume(volume)
```

These parameters, represented as floating-point numbers, enable flexible adjustments. For instance, setting a speed of 2.0 doubles the speed of speech, a pitch of 0.95 reduces pitch by about 5%, and a volume of 1.4 multiplies sound samples by 1.4 (with clipping if the maximum range of a 16-bit integer is exceeded).

Speech rate governs the speed of speech playback. A value of 2.0 results in a chipmunk-like, fast-paced speech, while 0.7 creates a slower, deliberate, and deeper tone, akin to a giant talking slowly. Adjust these parameters to tailor the audio output according to your application's requirements.

You may change the speed, pitch, rate, and volume parameters at any time, without having to flush or create a new sonic stream.

# Contributing

1. Fork it
2. Create your feature branch (git checkout -b my-new-feature)
3. Commit your changes (git commit -am 'Added some feature')
4. Push to the branch (git push origin my-new-feature)
5. Create new Pull Request

## Credits

The Sonic Go Library is a reimplementation of the original Sonic library by Bill Cox.  The [original Sonic library](https://github.com/waywardgeek/sonic.git), Copyright 2010, 2011, Bill Cox, is released under the Apache 2.0 license.

## Author

- Alexander Khudich
    - Email: alttagil (at) gmail.com
    - Twitter: [@alttagil](https://twitter.com/alttagil)

Feel free to explore the capabilities of the Sonic Go Library and incorporate it into your Go applications.