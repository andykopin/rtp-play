package wav

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"rtp-play/internal/codec"
)

const sampleRate = 8000

// Sources holds preloaded PCM payloads keyed by codec.
type Sources struct {
	Pcmu []byte
	Pcma []byte
}

func (s Sources) Get(c codec.Codec) []byte {
	switch c {
	case codec.PCMU:
		return s.Pcmu
	case codec.PCMA:
		return s.Pcma
	default:
		panic(fmt.Sprintf("unknown codec PT=%d", c.PayloadType))
	}
}

func LoadSources() (Sources, error) {
	mu, err := LoadWAV(codec.PCMU.File)
	if err != nil {
		return Sources{}, fmt.Errorf("PCMU: %w", err)
	}
	log.Printf("PCMU: loaded %d bytes (%.1fs)\n", len(mu), float64(len(mu))/sampleRate)

	ma, err := LoadWAV(codec.PCMA.File)
	if err != nil {
		return Sources{}, fmt.Errorf("PCMA: %w", err)
	}
	log.Printf("PCMA: loaded %d bytes (%.1fs)\n", len(ma), float64(len(ma))/sampleRate)

	return Sources{Pcmu: mu, Pcma: ma}, nil
}

func LoadWAV(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("close %s: %v", name, err)
		}
	}()
	return ReadWAVData(f)
}

// ReadWAVData walks RIFF chunks and returns the raw PCM payload from the "data" chunk.
func ReadWAVData(f io.ReadSeeker) ([]byte, error) {
	// RIFF header layout: 4 bytes "RIFF" | 4 bytes file size | 4 bytes "WAVE"
	hdr := make([]byte, 12)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return nil, err
	}
	if string(hdr[0:4]) != "RIFF" || string(hdr[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a WAV file")
	}

	// Each chunk starts with an 8-byte header: 4-byte ID + 4-byte body size (little-endian)
	chunk := make([]byte, 8)
	for {
		if _, err := io.ReadFull(f, chunk); err != nil {
			return nil, fmt.Errorf("data chunk not found")
		}
		// Body size of the current chunk in bytes
		size := binary.LittleEndian.Uint32(chunk[4:8])
		if string(chunk[0:4]) == "data" {
			payload := make([]byte, size)
			if _, err := io.ReadFull(f, payload); err != nil {
				return nil, err
			}
			return payload, nil
		}
		// Skip the body of unrecognized chunks (e.g. "fmt") to reach the next one;
		// whence=1 means seek relative to the current position
		if _, err := f.Seek(int64(size), 1); err != nil {
			return nil, err
		}
	}
}
