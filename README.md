# rtp-play

RTP PCMU stream example from Golang to ffplay.

## PCM basics

**Sample rate** — how many audio samples are captured per second. Telephony standard: **8000 Hz** (8 kHz). Each sample is one measurement of the sound wave amplitude.

**Bit depth** — bits used to represent one sample. G.711 uses **8 bits** per sample.

**Bitrate** — data rate of the audio stream:
```
bitrate = sample_rate × bit_depth × channels = 8000 × 8 × 1 = 64 000 bit/s = 64 kbps
```

**Frame** — a chunk of samples sent as one RTP packet. Standard for G.711: **20 ms**, which is:
```
frame_size = sample_rate × frame_duration = 8000 × 0.02 = 160 bytes
```

**Frame rate** — how many frames are sent per second:
```
frame_rate = 1000 ms / 20 ms = 50 packets/s
```

### PCMU vs PCMA

Both are G.711 variants: 8-bit, 8 kHz, mono. They differ only in the companding algorithm used to compress the linear PCM signal into 8 bits.

| | PCMU (µ-law) | PCMA (A-law) |
|---|---|---|
| Standard | G.711 µ-law | G.711 A-law |
| RTP payload type | **0** | **8** |
| Used in | North America, Japan | Europe, rest of world |
| Dynamic range | Better at low amplitudes | More uniform across range |

Conversion between PCMU and PCMA is lossless and done via a lookup table (256 entries).

---

## Usage

### 1. Start receiving with ffplay

```sh
ffplay -protocol_whitelist file,udp,rtp -i stream-pcma.sdp
```
or
```sh
ffplay -protocol_whitelist file,udp,rtp -i stream-pcmu.sdp
```

### 2. Send a test stream with ffmpeg

**1 kHz tone:**
```sh
ffmpeg -re -f lavfi -i "sine=frequency=1000:sample_rate=8000" \
    -ac 1 -c:a pcm_mulaw -f rtp rtp://127.0.0.1:5004
```

**White noise:**
```sh
ffmpeg -re -f lavfi -i "anoisesrc=color=white:sample_rate=8000" \
    -ac 1 -c:a pcm_mulaw -f rtp rtp://127.0.0.1:5004
```

**Pink noise:**
```sh
ffmpeg -re -f lavfi -i "anoisesrc=color=pink:sample_rate=8000" \
    -ac 1 -c:a pcm_mulaw -f rtp rtp://127.0.0.1:5004
```

**jazz.wav** (already PCMU, copy stream):
```sh
ffmpeg -re -i jazz.wav -c:a copy -f rtp rtp://127.0.0.1:5004
```

### 3. Stream jazz.wav from Go

```sh
go run ./cmd/rtp-play/
```

### Convert any source to G.711

Any input (mp3, aac, flac, video, etc.) → PCMU:
```sh
ffmpeg -i input.mp3 -ar 8000 -ac 1 -c:a pcm_mulaw output.wav
```

Any input → PCMA:
```sh
ffmpeg -i input.mp3 -ar 8000 -ac 1 -c:a pcm_alaw output.wav
```

Flags:
- `-ar 8000` — resample to 8 kHz
- `-ac 1` — downmix to mono
- `-c:a pcm_mulaw` / `-c:a pcm_alaw` — encode as PCMU / PCMA
