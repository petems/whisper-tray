package audio

import (
	"context"
	"fmt"

	"github.com/gordonklaus/portaudio"
	"github.com/petems/whisper-tray/internal/config"
)

type portAudioCapture struct {
	stream *portaudio.Stream
}

// New creates a new PortAudio-based audio capture
func New(cfg config.AudioConfig) (Capture, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize PortAudio: %w", err)
	}
	return &portAudioCapture{}, nil
}

func (p *portAudioCapture) Start(ctx context.Context, deviceID string, sampleRate int, out chan<- []float32) error {
	// Find device
	var device *portaudio.DeviceInfo
	if deviceID == "" {
		var err error
		device, err = portaudio.DefaultInputDevice()
		if err != nil {
			return fmt.Errorf("failed to get default input device: %w", err)
		}
	} else {
		devices, err := portaudio.Devices()
		if err != nil {
			return fmt.Errorf("failed to enumerate devices: %w", err)
		}
		for _, d := range devices {
			if d.Name != deviceID {
				continue
			}
			// Prefer entries that actually expose input channels (some devices appear twice for input/output)
			if d.MaxInputChannels > 0 {
				device = d
				break
			}
			// Keep as fallback but continue searching for an input-capable variant
			if device == nil {
				device = d
			}
		}
	}

	if device == nil {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	// Determine channel count; many USB mics (e.g., Yeti) expose stereo only, so respect their native input count.
	channels := int(device.MaxInputChannels)
	if channels <= 0 {
		return fmt.Errorf("device reports no input channels: %s", device.Name)
	}
	// Whisper expects mono, so we'll downmix later. Cap to 2 channels to keep buffer math simple.
	if channels > 2 {
		channels = 2
	}

	framesPerBuffer := 512
	// Allocate interleaved buffer sized for channel count.
	buffer := make([]float32, framesPerBuffer*channels)
	stream, err := portaudio.OpenStream(portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   device,
			Channels: channels,
			Latency:  device.DefaultLowInputLatency,
		},
		SampleRate:      float64(sampleRate),
		FramesPerBuffer: framesPerBuffer,
	}, buffer)

	if err != nil {
		return fmt.Errorf("failed to open audio stream: %w", err)
	}

	p.stream = stream

	if err := stream.Start(); err != nil {
		stream.Close()
		return fmt.Errorf("failed to start audio stream: %w", err)
	}

	// Read loop
	go func() {
		defer stream.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := stream.Read(); err != nil {
					return
				}
				// Copy buffer and downmix to mono if necessary before sending.
				monoSamples := downmixInterleaved(buffer, channels, framesPerBuffer)

				select {
				case out <- monoSamples:
				case <-ctx.Done():
					return
				default:
					// Drop if channel full (backpressure)
				}
			}
		}
	}()

	return nil
}

func (p *portAudioCapture) Stop() error {
	if p.stream != nil {
		return p.stream.Stop()
	}
	return nil
}

func (p *portAudioCapture) ListDevices() ([]AudioDevice, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	result := make([]AudioDevice, 0, len(devices))
	defaultDevice, _ := portaudio.DefaultInputDevice()

	for _, d := range devices {
		if d.MaxInputChannels > 0 {
			result = append(result, AudioDevice{
				ID:      d.Name,
				Name:    d.Name,
				Default: d == defaultDevice,
			})
		}
	}

	return result, nil
}

func (p *portAudioCapture) Close() error {
	if p.stream != nil {
		p.stream.Close()
	}
	portaudio.Terminate()
	return nil
}

// downmixInterleaved converts an interleaved multi-channel buffer into a mono slice.
// It allocates a fresh slice sized to the supplied frame count.
// Channels must be >= 1; when channels == 1 the first frames elements are copied directly.
func downmixInterleaved(buffer []float32, channels int, frames int) []float32 {
	mono := make([]float32, frames)
	if channels <= 1 {
		copy(mono, buffer[:frames])
		return mono
	}

	for frame := 0; frame < frames; frame++ {
		sum := float32(0)
		base := frame * channels
		for ch := 0; ch < channels; ch++ {
			sum += buffer[base+ch]
		}
		mono[frame] = sum / float32(channels)
	}
	return mono
}
