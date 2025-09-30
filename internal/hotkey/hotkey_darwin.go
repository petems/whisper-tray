//go:build darwin

package hotkey

/*
#cgo LDFLAGS: -framework Carbon
#include <Carbon/Carbon.h>

// Forward declaration for Go callback
extern void goHotkeyCallback(int pressed);

// Event handler for hotkeys
static OSStatus hotkeyHandler(EventHandlerCallRef nextHandler, EventRef theEvent, void* userData) {
    EventHotKeyID hkRef;
    GetEventParameter(theEvent, kEventParamDirectObject, typeEventHotKeyID, NULL, sizeof(hkRef), NULL, &hkRef);

    UInt32 eventKind = GetEventKind(theEvent);
    int pressed = (eventKind == kEventHotKeyPressed) ? 1 : 0;

    goHotkeyCallback(pressed);

    return noErr;
}

// Register hotkey with Carbon
static int registerHotkey(UInt32 keyCode, UInt32 modifiers) {
    EventTypeSpec eventTypes[2];
    eventTypes[0].eventClass = kEventClassKeyboard;
    eventTypes[0].eventKind = kEventHotKeyPressed;
    eventTypes[1].eventClass = kEventClassKeyboard;
    eventTypes[1].eventKind = kEventHotKeyReleased;

    EventHandlerUPP handlerUPP = NewEventHandlerUPP(hotkeyHandler);
    InstallApplicationEventHandler(handlerUPP, 2, eventTypes, NULL, NULL);

    EventHotKeyRef hotKeyRef;
    EventHotKeyID hotKeyID;
    hotKeyID.signature = 'htk1';
    hotKeyID.id = 1;

    OSStatus status = RegisterEventHotKey(keyCode, modifiers, hotKeyID, GetApplicationEventTarget(), 0, &hotKeyRef);

    return (status == noErr) ? 1 : 0;
}
*/
import "C"

import (
	"fmt"
)

type darwinManager struct {
	callback func(bool)
}

var globalManager *darwinManager

// New creates a new macOS hotkey manager using Carbon
func New() (Manager, error) {
	mgr := &darwinManager{}
	return mgr, nil
}

//export goHotkeyCallback
func goHotkeyCallback(pressed C.int) {
	if globalManager != nil && globalManager.callback != nil {
		globalManager.callback(pressed == 1)
	}
}

func (m *darwinManager) Register(accel string, callback func(pressed bool)) error {
	m.callback = callback
	globalManager = m

	// TODO: Parse accelerator string properly
	// For now: hardcoded to Control+Space on macOS
	// Space = keyCode 49
	// controlKey=0x1000 (cmdKey=0x100, shiftKey=0x200, optionKey=0x800)
	keyCode := C.UInt32(49)        // Space
	modifiers := C.UInt32(0x1000)  // Control key

	ret := C.registerHotkey(keyCode, modifiers)
	if ret == 0 {
		return fmt.Errorf("failed to register hotkey")
	}

	return nil
}

func (m *darwinManager) Unregister(accel string) error {
	// TODO: UnregisterEventHotKey implementation
	return nil
}

func (m *darwinManager) Close() error {
	globalManager = nil
	return nil
}