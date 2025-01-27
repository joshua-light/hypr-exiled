package sound

import (
	"bytes"
	"embed"
	"fmt"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/wav"
)

type SoundNotifier struct {
	assets embed.FS
}

func NewSoundNotifier(assets embed.FS) (*SoundNotifier, error) {
	// Try to initialize speaker, handle "already initialized" case
	err := speaker.Init(44100, 4096)
	if err != nil {
		if err.Error() == "speaker cannot be initialized more than once" {
			// Speaker already initialized, safe to proceed
			return &SoundNotifier{assets: assets}, nil
		}
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

	// Decode directly from memory
	streamer, _, err := wav.Decode(bytes.NewReader(soundData))
	if err != nil {
		return fmt.Errorf("failed to decode WAV: %w", err)
	}
	defer streamer.Close()

	// Channel to wait for playback completion
	done := make(chan struct{})

	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		close(done)
	})))

	// Block until playback finishes
	<-done
	return nil
}

func (s *SoundNotifier) Close() error {
	speaker.Close()
	return nil
}
