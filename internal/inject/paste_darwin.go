//go:build darwin

package inject

/*
#cgo LDFLAGS: -framework ApplicationServices -framework Carbon
#include <ApplicationServices/ApplicationServices.h>
#include <Carbon/Carbon.h>

// Send Cmd+V paste shortcut
void sendPasteShortcut() {
    CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);

    // Press Cmd+V
    CGEventRef cmdDown = CGEventCreateKeyboardEvent(source, (CGKeyCode)55, true); // Cmd key
    CGEventSetFlags(cmdDown, kCGEventFlagMaskCommand);
    CGEventRef vDown = CGEventCreateKeyboardEvent(source, (CGKeyCode)9, true); // V key
    CGEventSetFlags(vDown, kCGEventFlagMaskCommand);

    // Release V+Cmd
    CGEventRef vUp = CGEventCreateKeyboardEvent(source, (CGKeyCode)9, false);
    CGEventRef cmdUp = CGEventCreateKeyboardEvent(source, (CGKeyCode)55, false);

    // Post events
    CGEventPost(kCGHIDEventTap, cmdDown);
    CGEventPost(kCGHIDEventTap, vDown);
    CGEventPost(kCGHIDEventTap, vUp);
    CGEventPost(kCGHIDEventTap, cmdUp);

    CFRelease(cmdDown);
    CFRelease(vDown);
    CFRelease(vUp);
    CFRelease(cmdUp);
    CFRelease(source);
}
*/
import "C"

import (
	"context"
	"fmt"
	"time"

	"github.com/atotto/clipboard"
)

// sendPasteShortcut sends Cmd+V on macOS
func sendPasteShortcut() error {
	C.sendPasteShortcut()
	return nil
}

// platformPaste implements clipboard-paste strategy for macOS
func platformPaste(ctx context.Context, text string) error {
	// Save current clipboard
	oldClip, err := clipboard.ReadAll()
	if err != nil {
		oldClip = "" // If clipboard read fails, proceed anyway
	}

	// Set new clipboard
	if err := clipboard.WriteAll(text); err != nil {
		return fmt.Errorf("failed to write clipboard: %w", err)
	}

	// Small delay to ensure clipboard is set
	time.Sleep(50 * time.Millisecond)

	// Send Cmd+V
	if err := sendPasteShortcut(); err != nil {
		return fmt.Errorf("failed to send paste shortcut: %w", err)
	}

	// Wait a bit for paste to complete
	time.Sleep(100 * time.Millisecond)

	// Restore old clipboard (best effort)
	// Check if user hasn't changed it in the meantime
	currentClip, _ := clipboard.ReadAll()
	if currentClip == text {
		clipboard.WriteAll(oldClip)
	}

	return nil
}