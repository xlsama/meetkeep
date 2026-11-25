package audio

import (
	"encoding/binary"
	"math"
)

// CaptureRate is the sample rate of what the capture module records.
const CaptureRate = 48000

// Samples decodes 16-bit little-endian PCM.
func Samples(pcm []byte) []int16 {
	out := make([]int16, len(pcm)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(pcm[2*i:]))
	}
	return out
}

// Bytes encodes samples as 16-bit little-endian PCM.
func Bytes(samples []int16) []byte {
	out := make([]byte, 2*len(samples))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(out[2*i:], uint16(s))
	}
	return out
}

// Level maps the loudness of samples to 0..1, from -50 dBFS to full scale.
func Level(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		v := float64(s) / 32768
		sum += v * v
	}
	rms := math.Sqrt(sum / float64(len(samples)))
	if rms <= 0 {
		return 0
	}
	return min(max((20*math.Log10(rms)+50)/50, 0), 1)
}

// Quietest returns where in 16 kHz samples the quietest pause of pauseMs
// starts, in milliseconds: the one nearest to their middle when several are
// as quiet.
func Quietest(samples []int16, pauseMs int) int {
	const frame = SampleRate / 50
	frames := len(samples) / frame
	span := pauseMs / 20
	if frames <= span {
		return 0
	}
	energy := make([]float64, frames)
	for i := range energy {
		for _, s := range samples[i*frame : (i+1)*frame] {
			energy[i] += float64(s) * float64(s)
		}
	}
	var sum float64
	for _, e := range energy[:span] {
		sum += e
	}
	middle := (frames - span) / 2
	distance := func(i int) int { return max(i-middle, middle-i) }
	best, least := 0, sum
	for i := 1; i+span <= frames; i++ {
		sum += energy[i+span-1] - energy[i-1]
		if sum < least || sum == least && distance(i) < distance(best) {
			best, least = i, sum
		}
	}
	return best * 20
}

const decimation = CaptureRate / SampleRate

var lowpass = func() []float32 {
	const n = 63
	cutoff := 7200.0 / CaptureRate
	taps := make([]float32, n)
	var sum float64
	for i := range taps {
		x := float64(i) - float64(n-1)/2
		v := 2 * cutoff
		if x != 0 {
			v = math.Sin(2*math.Pi*cutoff*x) / (math.Pi * x)
		}
		v *= 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(n-1))
		taps[i] = float32(v)
		sum += v
	}
	for i := range taps {
		taps[i] /= float32(sum)
	}
	return taps
}()

// Decimator turns a stream of 48 kHz audio into 16 kHz.
type Decimator struct {
	buffer []float32
}

// Push takes the next samples and returns what they complete.
func (d *Decimator) Push(samples []int16) []int16 {
	for _, s := range samples {
		d.buffer = append(d.buffer, float32(s))
	}
	out := make([]int16, 0, len(samples)/decimation+1)
	at := 0
	for ; at+len(lowpass) <= len(d.buffer); at += decimation {
		var sum float32
		for k, tap := range lowpass {
			sum += tap * d.buffer[at+k]
		}
		out = append(out, int16(min(max(sum, math.MinInt16), math.MaxInt16)))
	}
	d.buffer = append(d.buffer[:0], d.buffer[at:]...)
	return out
}
