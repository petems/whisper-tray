---

# Additional Go files

# internal/hotkey/hotkey_linux.go
//go:build linux

package hotkey

/*
#cgo pkg-config: x11 xtst
#include <X11/Xlib.h>
#include <X11/keysym.h>
#include <X11/extensions/XTest.h>
#include <stdlib.h>

Display* displayPtr = NULL;

int grabKey(int keycode, int modifiers) {
    if (displayPtr == NULL) {
        displayPtr = XOpenDisplay(NULL);
    }
    if (displayPtr == NULL) return 0;
    
    Window root = DefaultRootWindow(displayPtr);
    XGrabKey(displayPtr, keycode, modifiers, root, False, GrabModeAsync, GrabModeAsync);
    XSelectInput(displayPtr, root, KeyPressMask | KeyReleaseMask);
    XSync(displayPtr, False);
    
    return 1;
}

int checkEvent(int* keycode, int* pressed) {
    if (displayPtr == NULL) return 0;
    
    XEvent event;
    if (XPending(displayPtr) > 0) {
        XNextEvent(displayPtr, &event);
        if (event.type == KeyPress || event.type == KeyRelease) {
            *keycode = event.xkey.keycode;
            *pressed = (event.type == KeyPress) ? 1 : 0;
            return 1;
        }
    }
    return 0;
}
*/
import "C"

import (
	"fmt"
	"time"
)

type linuxManager struct {
	callbacks map[int]func(bool)
	stop      chan struct{}
}

func New() (Manager, error) {
	mgr := &linuxManager{
		callbacks: make(map[int]func(bool)),
		stop:      make(chan struct{}),
	}
	
	go mgr.eventLoop()
	
	return mgr, nil
}

func (m *linuxManager) Register(accel string, callback func(pressed bool)) error {
	// Parse "Alt+Space" -> keycode 65, Mod1Mask (Alt)
	keycode := 65 // Space
	modifiers := 8 // Mod1Mask (Alt)
	
	ret := C.grabKey(C.int(keycode), C.int(modifiers))
	if ret == 0 {
		return fmt.Errorf("failed to grab key")
	}
	
	m.callbacks[keycode] = callback
	return nil
}

func (m *linuxManager) eventLoop() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			var keycode, pressed C.int
			if C.checkEvent(&keycode, &pressed) != 0 {
				if cb, ok := m.callbacks[int(keycode)]; ok {
					cb(pressed == 1)
				}
			}
		}
	}
}

func (m *linuxManager) Unregister(accel string) error {
	// XUngrabKey would go here
	return nil
}

func (m *linuxManager) Close() error {
	close(m.stop)
	return nil
}

---

# internal/logging/logging.go
package logging

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func New() zerolog.Logger {
	logPath := getLogPath()
	
	// Ensure directory exists
	os.MkdirAll(filepath.Dir(logPath), 0755)
	
	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open log file")
	}
	
	// Multi-writer: console + file
	multi := zerolog.MultiLevelWriter(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
		logFile,
	)
	
	return zerolog.New(multi).With().Timestamp().Caller().Logger()
}

func getLogPath() string {
	var base string
	
	switch runtime.GOOS {
	case "darwin":
		base = os.Getenv("HOME") + "/Library/Logs"
	case "windows":
		base = os.Getenv("LOCALAPPDATA")
	default:
		if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
			base = xdg
		} else {
			base = os.Getenv("HOME") + "/.local/state"
		}
	}
	
	return filepath.Join(base, "whisper-tray", "whisper-tray.log")
}

---

# internal/permissions/permissions_darwin.go
//go:build darwin

package permissions

/*
#cgo LDFLAGS: -framework AVFoundation -framework Cocoa
#import <AVFoundation/AVFoundation.h>
#import <Cocoa/Cocoa.h>

int checkMicrophonePermission() {
    AVAuthorizationStatus status = [AVCaptureDevice authorizationStatusForMediaType:AVMediaTypeAudio];
    return (int)status;
}

void requestMicrophonePermission() {
    [AVCaptureDevice requestAccessForMediaType:AVMediaTypeAudio completionHandler:^(BOOL granted) {}];
}

int checkAccessibilityPermission() {
    NSDictionary *options = @{(__bridge id)kAXTrustedCheckOptionPrompt: @YES};
    return AXIsProcessTrustedWithOptions((__bridge CFDictionaryRef)options) ? 1 : 0;
}
*/
import "C"

import "fmt"

const (
	PermissionNotDetermined = 0
	PermissionRestricted    = 1
	PermissionDenied        = 2
	PermissionAuthorized    = 3
)

func CheckMicrophone() (int, error) {
	status := int(C.checkMicrophonePermission())
	return status, nil
}

func RequestMicrophone() error {
	C.requestMicrophonePermission()
	return nil
}

func CheckAccessibility() (bool, error) {
	status := int(C.checkAccessibilityPermission())
	return status == 1, nil
}

func PromptAccessibility() error {
	// Prompt shown by checkAccessibilityPermission with kAXTrustedCheckOptionPrompt
	return nil
}

func EnsurePermissions() error {
	// Check microphone
	micStatus, _ := CheckMicrophone()
	if micStatus != PermissionAuthorized {
		fmt.Println("⚠️  Microphone permission required")
		RequestMicrophone()
		return fmt.Errorf("microphone permission not granted")
	}
	
	// Check accessibility
	axGranted, _ := CheckAccessibility()
	if !axGranted {
		fmt.Println("⚠️  Accessibility permission required for hotkeys")
		fmt.Println("   Go to: System Settings → Privacy & Security → Accessibility")
		PromptAccessibility()
		return fmt.Errorf("accessibility permission not granted")
	}
	
	return nil
}

---

# resources/Info.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>whisper-tray</string>
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
    <key>CFBundleIdentifier</key>
    <string>com.whispertray.app</string>
    <key>CFBundleName</key>
    <string>WhisperTray</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>
    <key>CFBundleVersion</key>
    <string>1</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    <key>LSUIElement</key>
    <true/>
    <key>NSMicrophoneUsageDescription</key>
    <string>WhisperTray needs microphone access to transcribe your voice.</string>
</dict>
</plist>

---

# scripts/install-linux.sh
#!/bin/bash
# Linux installation script

set -e

echo "Installing WhisperTray..."

# Detect package manager
if command -v apt-get &> /dev/null; then
    sudo apt-get update
    sudo apt-get install -y portaudio19-dev libx11-dev libxtst-dev
elif command -v dnf &> /dev/null; then
    sudo dnf install -y portaudio-devel libX11-devel libXtst-devel
elif command -v pacman &> /dev/null; then
    sudo pacman -S --noconfirm portaudio libx11 libxtst
fi

# Copy binary
sudo cp whisper-tray /usr/local/bin/
sudo chmod +x /usr/local/bin/whisper-tray

# Create desktop entry
mkdir -p ~/.local/share/applications
cat > ~/.local/share/applications/whisper-tray.desktop <<EOF
[Desktop Entry]
Type=Application
Name=WhisperTray
Comment=Local voice dictation
Exec=/usr/local/bin/whisper-tray
Icon=whisper-tray
Terminal=false
Categories=Utility;
EOF

echo "✓ Installation complete!"
echo "Run 'whisper-tray' to start"

---

# Test file example
# internal/app/app_test.go
package app

import (
	"context"
	"testing"
	"time"
)

type mockCapture struct{}

func (m *mockCapture) Start(ctx context.Context, deviceID string, sampleRate int, out chan<- []float32) error {
	// Send test audio
	go func() {
		samples := make([]float32, 512)
		for i := 0; i < 10; i++ {
			select {
			case out <- samples:
			case <-ctx.Done():
				return
			}
			time.Sleep(32 * time.Millisecond)
		}
	}()
	return nil
}

func (m *mockCapture) Stop() error                            { return nil }
func (m *mockCapture) ListDevices() ([]AudioDevice, error)   { return nil, nil }
func (m *mockCapture) Close() error                          { return nil }

func TestDictationFlow(t *testing.T) {
	// Test push-to-talk flow
	// This would test the full cycle: hotkey → start → audio → whisper → inject
	
	// For now, basic structure test
	cfg := &config.Config{
		Mode: "PushToTalk",
	}
	
	app := New(Config{
		Audio:  &mockCapture{},
		Config: cfg,
	})
	
	if app == nil {
		t.Fatal("Failed to create app")
	}
}