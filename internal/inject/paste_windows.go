//go:build windows

package inject

import (
	"context"
	"fmt"
)

// platformPaste implements clipboard-paste strategy for Windows
// TODO: Implement using Win32 API (SetClipboardData + SendInput for Ctrl+V)
func platformPaste(ctx context.Context, text string) error {
	return fmt.Errorf("paste not yet implemented on Windows")
}

// platformType implements keyboard typing for Windows
// TODO: Implement using Win32 SendInput API
func platformType(ctx context.Context, text string) error {
	return fmt.Errorf("type not yet implemented on Windows")
}