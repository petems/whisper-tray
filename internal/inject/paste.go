package inject

import (
	"context"
	"fmt"

	"github.com/petems/whisper-tray/internal/config"
)

type pasteInjector struct {
	cfg config.InjectConfig
}

// New creates a new text injector
func New(cfg config.InjectConfig) Injector {
	return &pasteInjector{
		cfg: cfg,
	}
}

// Paste injects text using clipboard + paste shortcut
// Implementation is platform-specific (see paste_darwin.go, paste_linux.go, etc.)
func (p *pasteInjector) Paste(ctx context.Context, text string) error {
	return platformPaste(ctx, text)
}

// Type injects text using keyboard simulation
func (p *pasteInjector) Type(ctx context.Context, text string) error {
	// TODO: Platform-specific keystroke injection
	// - Windows: SendInput
	// - macOS: CGEvent
	// - Linux: XTest
	return fmt.Errorf("Type not yet implemented")
}

// PasteOrType tries paste first, falls back to type if needed
func (p *pasteInjector) PasteOrType(ctx context.Context, text string) error {
	if p.cfg.PreferPaste {
		if err := p.Paste(ctx, text); err == nil {
			return nil
		}
	}
	return p.Type(ctx, text)
}