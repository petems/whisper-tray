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

// platformType implements keyboard typing for macOS using CGEvent
func platformType(ctx context.Context, text string) error {
	// Convert string to uint16 array (UTF-16) for CGEvent
	utf16 := []uint16{}
	for _, r := range text {
		if r <= 0xFFFF {
			utf16 = append(utf16, uint16(r))
		} else {
			// Handle surrogate pairs for characters outside BMP
			r -= 0x10000
			utf16 = append(utf16, uint16((r>>10)+0xD800))
			utf16 = append(utf16, uint16((r&0x3FF)+0xDC00))
		}
	}

	for i, char := range utf16 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Create keyboard event for this character
		source := C.CGEventSourceCreate(C.kCGEventSourceStateHIDSystemState)
		if source == 0 {
			return fmt.Errorf("failed to create event source")
		}

		// Create a keyboard event with virtual key 0
		event := C.CGEventCreateKeyboardEvent(source, 0, true)
		if event == 0 {
			C.CFRelease(C.CFTypeRef(source))
			return fmt.Errorf("failed to create keyboard event")
		}

		// Set the Unicode string for this event
		unichar := C.UniChar(char)
		C.CGEventKeyboardSetUnicodeString(event, 1, &unichar)

		// Post key down
		C.CGEventPost(C.kCGHIDEventTap, event)

		// Create key up event
		C.CGEventSetType(event, C.kCGEventKeyUp)
		C.CGEventPost(C.kCGHIDEventTap, event)

		C.CFRelease(C.CFTypeRef(event))
		C.CFRelease(C.CFTypeRef(source))

		// Small delay between keystrokes for natural typing
		// Skip delay for last character
		if i < len(utf16)-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	return nil
}