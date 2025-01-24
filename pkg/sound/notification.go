package sound

import (
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/wav"
)

type SoundNotifier struct {
	assets embed.FS
}

func NewSoundNotifier(assets embed.FS) (*SoundNotifier, error) {
	// Initialize speaker with default settings
	if err := speaker.Init(44100, 4096); err != nil {
		return nil, fmt.Errorf("failed to initialize audio: %w", err)
	}

	return &SoundNotifier{
		assets: assets,
	}, nil
}

func (s *SoundNotifier) PlayTradeSound() error {
	soundData, err := s.assets.ReadFile("assets/trade.wav")
	if err != nil {
		return fmt.Errorf("failed to read sound file: %w", err)
	}

	// Create temporary file to read WAV data
	tmpFile, err := os.CreateTemp("", "trade-*.wav")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(soundData); err != nil {
		return fmt.Errorf("failed to write sound data: %w", err)
	}
	tmpFile.Close()

	// Open and decode the WAV file
	f, err := os.Open(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to open temp file: %w", err)
	}
	defer f.Close()

	streamer, _, err := wav.Decode(f)
	if err != nil {
		return fmt.Errorf("failed to decode WAV: %w", err)
	}
	defer streamer.Close()

	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		streamer.Close()
	})))

	// Wait for sound to finish playing
	time.Sleep(time.Second)
	return nil
}
