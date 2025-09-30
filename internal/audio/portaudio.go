package audio

import (
	"context"
	"fmt"

	"github.com/petems/whisper-tray/internal/config"
	"github.com/gordonklaus/portaudio"
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
			if d.Name == deviceID {
				device = d
				break
			}
		}
	}

	if device == nil {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	// Open stream: mono, specified sample rate, float32
	buffer := make([]float32, 512)
	stream, err := portaudio.OpenStream(portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   device,
			Channels: 1,
			Latency:  device.DefaultLowInputLatency,
		},
		SampleRate:      float64(sampleRate),
		FramesPerBuffer: len(buffer),
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
				// Copy buffer and send
				samples := make([]float32, len(buffer))
				copy(samples, buffer)

				select {
				case out <- samples:
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