package codec

// Codec identifies a G.711 encoding variant and carries its RTP payload type.
type Codec struct {
	// PayloadType is the static PT number assigned by RFC 3551.
	PayloadType uint8
	// File is the WAV source file for this codec.
	File string
	// SampleRate is the number of samples per second.
	SampleRate int
	// FrameSizeMs is the duration of one RTP frame in milliseconds.
	FrameSizeMs int
}

// FrameSize returns the number of samples per frame.
func (c Codec) FrameSize() int {
	return c.SampleRate * c.FrameSizeMs / 1000
}

var (
	// PCMU is G.711 µ-law (PT=0, North America / Japan)
	PCMU = Codec{PayloadType: 0, File: "jazz-pcmu.wav", SampleRate: 8000, FrameSizeMs: 20}

	// PCMA is G.711 A-law (PT=8, Europe / rest of the world)
	PCMA = Codec{PayloadType: 8, File: "jazz-pcma.wav", SampleRate: 8000, FrameSizeMs: 20}
)
