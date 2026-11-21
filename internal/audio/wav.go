// Package audio cuts speech into utterances and writes recordings.
package audio

import "encoding/binary"

const SampleRate = 16000

type header struct {
	RIFF       [4]byte
	Size       uint32
	WAVE       [4]byte
	Fmt        [4]byte
	FmtSize    uint32
	Format     uint16
	Channels   uint16
	Rate       uint32
	ByteRate   uint32
	BlockAlign uint16
	Bits       uint16
	Data       [4]byte
	DataSize   uint32
}

func newHeader(size uint32) header {
	return header{
		RIFF: [4]byte{'R', 'I', 'F', 'F'}, Size: 36 + size,
		WAVE: [4]byte{'W', 'A', 'V', 'E'}, Fmt: [4]byte{'f', 'm', 't', ' '},
		FmtSize: 16, Format: 1, Channels: 1,
		Rate: SampleRate, ByteRate: SampleRate * 2, BlockAlign: 2, Bits: 16,
		Data: [4]byte{'d', 'a', 't', 'a'}, DataSize: size,
	}
}

// EncodeWAV returns pcm as a 16 kHz mono WAV file.
func EncodeWAV(pcm []int16) []byte {
	out := make([]byte, 0, 44+len(pcm)*2)
	out, _ = binary.Append(out, binary.LittleEndian, newHeader(uint32(len(pcm)*2)))
	out, _ = binary.Append(out, binary.LittleEndian, pcm)
	return out
}
