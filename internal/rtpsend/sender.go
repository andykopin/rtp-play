package rtpsend

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"rtp-play/internal/codec"
	"rtp-play/internal/wav"

	"github.com/pion/rtp"
)

// RunStream orchestrates source loading, connection setup, and streaming.
func RunStream(address string, timeout time.Duration, codec codec.Codec) error {
	src, err := wav.LoadSources()
	if err != nil {
		return err
	}

	conn, err := DialUDP(address)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("close connection: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Printf("Streaming to %s ...\n", address)

	if err := SendFrames(ctx, conn, src.Get(codec), codec); err != nil {
		return err
	}

	if ctx.Err() != nil {
		log.Println("Stream timed out.")
	} else {
		log.Println("Done.")
	}
	return nil
}

func DialUDP(addr string) (net.Conn, error) {
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial udp %s: %w", addr, err)
	}
	return conn, nil
}

// SendFrames sends PCM data as RTP packets paced by a 20ms ticker until data is exhausted or ctx is cancelled.
func SendFrames(ctx context.Context, conn net.Conn, data []byte, codec codec.Codec) error {
	const ssrc uint32 = 0xDEADBEEF // arbitrary synchronization source identifier

	var seqNum uint16
	var timestamp uint32

	frameSize := codec.FrameSize()

	ticker := time.NewTicker(time.Duration(codec.FrameSizeMs) * time.Millisecond)
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

		// Marshal serializes the packet into the binary wire format defined by RFC 3550
		buf, err := pkt.Marshal()
		if err != nil {
			return fmt.Errorf("marshal rtp: %w", err)
		}
		if _, err = conn.Write(buf); err != nil {
			return fmt.Errorf("write udp: %w", err)
		}

		seqNum++
		// Timestamp advances by the number of samples in the frame, not by milliseconds
		timestamp += uint32(frameSize)
		offset = end
	}
}
