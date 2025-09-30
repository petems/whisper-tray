package inject

import (
	"context"

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
	// Platform-specific implementation
	// - macOS: CGEvent (see paste_darwin.go)
	// - Linux: XTest (TODO)
	// - Windows: SendInput (TODO)
	return platformType(ctx, text)
}

// PasteOrType tries paste first, falls back to type if needed
func (p *pasteInjector) PasteOrType(ctx context.Context, text string) error {
	// Try paste first if preferred or if Type is not implemented
	if p.cfg.PreferPaste {
		return p.Paste(ctx, text)
	}

	// Try Type, fall back to Paste if not implemented
	if err := p.Type(ctx, text); err != nil {
		// If Type fails (e.g., not implemented), use Paste
		return p.Paste(ctx, text)
	}
	return nil
}