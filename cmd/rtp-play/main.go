package main

import (
	"log"

	"rtp-play/internal/codec"
	"rtp-play/internal/config"
	"rtp-play/internal/rtpsend"
)

func main() {
	cfg := config.Load()
	if err := rtpsend.RunStream(cfg.Addr, cfg.Timeout, codec.PCMA); err != nil {
		log.Fatal(err)
	}
}
