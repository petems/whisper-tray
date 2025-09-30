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

// CheckMicrophone returns the current microphone permission status
func CheckMicrophone() (int, error) {
	status := int(C.checkMicrophonePermission())
	return status, nil
}

// RequestMicrophone triggers the system microphone permission dialog
func RequestMicrophone() error {
	C.requestMicrophonePermission()
	return nil
}

// CheckAccessibility checks if the app has accessibility permissions (needed for hotkeys)
func CheckAccessibility() (bool, error) {
	status := int(C.checkAccessibilityPermission())
	return status == 1, nil
}

// PromptAccessibility prompts for accessibility permissions
func PromptAccessibility() error {
	// Prompt shown by checkAccessibilityPermission with kAXTrustedCheckOptionPrompt
	return nil
}

// EnsurePermissions checks and requests all required permissions
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