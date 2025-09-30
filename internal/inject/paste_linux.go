//go:build linux

package inject

import (
	"context"
	"fmt"
)

// platformPaste implements clipboard-paste strategy for Linux
// TODO: Implement using XTest/xdotool or Wayland protocols
func platformPaste(ctx context.Context, text string) error {
	return fmt.Errorf("paste not yet implemented on Linux")
}

// platformType implements keyboard typing for Linux
// TODO: Implement using XTest (X11) or appropriate Wayland method
func platformType(ctx context.Context, text string) error {
	return fmt.Errorf("type not yet implemented on Linux")
}