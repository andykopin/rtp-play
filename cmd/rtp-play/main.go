package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/pion/rtp"
)

const (
	sampleRate      = 8000
	frameSizeMs     = 20
	frameSize       = sampleRate * frameSizeMs / 1000 // 160 bytes per 20ms frame (8000 samples/s × 0.02s)
	streamTimeoutMs = 10_000                          // timeout for the stream goroutine in milliseconds
)

// Codec identifies a G.711 encoding variant and carries its RTP payload type.
type Codec struct {
	// PayloadType is the static PT number assigned by RFC 3551.
	PayloadType uint8
	// File is the WAV source file for this codec.
	File string
}

var (
	// PCMU is G.711 µ-law (PT=0, North America / Japan)
	PCMU = Codec{PayloadType: 0, File: "jazz-pcmu.wav"}
	// PCMA is G.711 A-law (PT=8, Europe / rest of world)
	PCMA = Codec{PayloadType: 8, File: "jazz-pcma.wav"}
)

// sources holds pre-loaded PCM payloads keyed by codec.
type sources struct {
	pcmu []byte
	pcma []byte
}

func (s sources) get(c Codec) []byte {
	switch c {
	case PCMU:
		return s.pcmu
	case PCMA:
		return s.pcma
	default:
		panic(fmt.Sprintf("unknown codec PT=%d", c.PayloadType))
	}
}

func main() {
	if err := run("127.0.0.1:5004", PCMA); err != nil {
		log.Fatal(err)
	}
}

// run orchestrates source loading, connection setup, and streaming.
func run(addr string, codec Codec) error {
	src, err := loadSources()
	if err != nil {
		return err
	}

	conn, err := dialUDP(addr)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("close conn: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), streamTimeoutMs*time.Millisecond)
	defer cancel()

	fmt.Printf("Streaming to %s ...\n", addr)

	if err := sendFrames(ctx, conn, src.get(codec), codec); err != nil {
		return err
	}

	if ctx.Err() != nil {
		fmt.Println("Stream timed out.")
	} else {
		fmt.Println("Done.")
	}
	return nil
}

// dialUDP opens a connected UDP socket to addr.
func dialUDP(addr string) (net.Conn, error) {
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial udp %s: %w", addr, err)
	}
	return conn, nil
}

// sendFrames sends PCM data as RTP packets paced by a 20ms ticker until data is exhausted or ctx is cancelled.
func sendFrames(ctx context.Context, conn net.Conn, data []byte, codec Codec) error {
	const ssrc uint32 = 0xDEADBEEF // arbitrary synchronization source identifier

	var seqNum uint16
	var timestamp uint32

	ticker := time.NewTicker(frameSizeMs * time.Millisecond)
	defer ticker.Stop()

	for offset := 0; ; {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}

		end := offset + frameSize
		// Stop when there is less than one full frame remaining
		if end > len(data) {
			return nil
		}

		pkt := &rtp.Packet{
			Header: rtp.Header{
				Version:        2, // RTP version 2 is the only version in use (RFC 3550)
				PayloadType:    codec.PayloadType,
				SequenceNumber: seqNum,
				Timestamp:      timestamp,
				SSRC:           ssrc,
			},
			Payload: data[offset:end],
		}

		// Marshal serialises the packet into the binary wire format defined by RFC 3550
		buf, err := pkt.Marshal()
		if err != nil {
			return fmt.Errorf("marshal rtp: %w", err)
		}
		if _, err = conn.Write(buf); err != nil {
			return fmt.Errorf("write udp: %w", err)
		}

		seqNum++
		// Timestamp advances by the number of samples in the frame, not by milliseconds
		timestamp += frameSize
		offset = end
	}
}

func loadSources() (sources, error) {
	mu, err := loadWAV(PCMU.File)
	if err != nil {
		return sources{}, fmt.Errorf("PCMU: %w", err)
	}
	fmt.Printf("PCMU: loaded %d bytes (%.1fs)\n", len(mu), float64(len(mu))/sampleRate)

	ma, err := loadWAV(PCMA.File)
	if err != nil {
		return sources{}, fmt.Errorf("PCMA: %w", err)
	}
	fmt.Printf("PCMA: loaded %d bytes (%.1fs)\n", len(ma), float64(len(ma))/sampleRate)

	return sources{pcmu: mu, pcma: ma}, nil
}

func loadWAV(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("close %s: %v", name, err)
		}
	}()
	return readWAVData(f)
}

// readWAVData walks RIFF chunks and returns the raw PCM payload from the "data" chunk.
func readWAVData(f *os.File) ([]byte, error) {
	// RIFF header layout: 4 bytes "RIFF" | 4 bytes file size | 4 bytes "WAVE"
	hdr := make([]byte, 12)
	if _, err := f.Read(hdr); err != nil {
		return nil, err
	}
	if string(hdr[0:4]) != "RIFF" || string(hdr[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a WAV file")
	}

	// Each chunk starts with an 8-byte header: 4-byte ID + 4-byte body size (little-endian)
	chunk := make([]byte, 8)
	for {
		if _, err := f.Read(chunk); err != nil {
			return nil, fmt.Errorf("data chunk not found")
		}
		// Body size of the current chunk in bytes
		size := binary.LittleEndian.Uint32(chunk[4:8])
		if string(chunk[0:4]) == "data" {
			payload := make([]byte, size)
			if _, err := f.Read(payload); err != nil {
				return nil, err
			}
			return payload, nil
		}
		// Skip the body of unrecognised chunks (e.g. "fmt ") to reach the next one;
		// whence=1 means seek relative to the current position
		if _, err := f.Seek(int64(size), 1); err != nil {
			return nil, err
		}
	}
}
