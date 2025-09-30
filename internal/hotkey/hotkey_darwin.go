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

// Hotkey registration state (stored for cleanup)
typedef struct {
    EventHandlerRef handlerRef;
    EventHandlerUPP handlerUPP;
    EventHotKeyRef hotKeyRef;
    int registered;
} HotkeyState;

static HotkeyState gHotkeyState = {NULL, NULL, NULL, 0};

// Register hotkey with Carbon
static int registerHotkey(UInt32 keyCode, UInt32 modifiers) {
    // Clean up existing registration if any
    if (gHotkeyState.registered) {
        if (gHotkeyState.hotKeyRef) {
            UnregisterEventHotKey(gHotkeyState.hotKeyRef);
            gHotkeyState.hotKeyRef = NULL;
        }
        if (gHotkeyState.handlerRef) {
            RemoveEventHandler(gHotkeyState.handlerRef);
            gHotkeyState.handlerRef = NULL;
        }
        if (gHotkeyState.handlerUPP) {
            DisposeEventHandlerUPP(gHotkeyState.handlerUPP);
            gHotkeyState.handlerUPP = NULL;
        }
        gHotkeyState.registered = 0;
    }

    EventTypeSpec eventTypes[2];
    eventTypes[0].eventClass = kEventClassKeyboard;
    eventTypes[0].eventKind = kEventHotKeyPressed;
    eventTypes[1].eventClass = kEventClassKeyboard;
    eventTypes[1].eventKind = kEventHotKeyReleased;

    gHotkeyState.handlerUPP = NewEventHandlerUPP(hotkeyHandler);
    OSStatus status = InstallApplicationEventHandler(
        gHotkeyState.handlerUPP,
        2,
        eventTypes,
        NULL,
        &gHotkeyState.handlerRef
    );

    if (status != noErr) {
        DisposeEventHandlerUPP(gHotkeyState.handlerUPP);
        gHotkeyState.handlerUPP = NULL;
        return 0;
    }

    EventHotKeyID hotKeyID;
    hotKeyID.signature = 'htk1';
    hotKeyID.id = 1;

    status = RegisterEventHotKey(
        keyCode,
        modifiers,
        hotKeyID,
        GetApplicationEventTarget(),
        0,
        &gHotkeyState.hotKeyRef
    );

    if (status != noErr) {
        RemoveEventHandler(gHotkeyState.handlerRef);
        DisposeEventHandlerUPP(gHotkeyState.handlerUPP);
        gHotkeyState.handlerRef = NULL;
        gHotkeyState.handlerUPP = NULL;
        gHotkeyState.hotKeyRef = NULL;
        return 0;
    }

    gHotkeyState.registered = 1;
    return 1;
}

// Unregister and cleanup hotkey resources
static void unregisterHotkey() {
    if (!gHotkeyState.registered) {
        return;
    }

    if (gHotkeyState.hotKeyRef) {
        UnregisterEventHotKey(gHotkeyState.hotKeyRef);
        gHotkeyState.hotKeyRef = NULL;
    }

    if (gHotkeyState.handlerRef) {
        RemoveEventHandler(gHotkeyState.handlerRef);
        gHotkeyState.handlerRef = NULL;
    }

    if (gHotkeyState.handlerUPP) {
        DisposeEventHandlerUPP(gHotkeyState.handlerUPP);
        gHotkeyState.handlerUPP = NULL;
    }

    gHotkeyState.registered = 0;
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
	C.unregisterHotkey()
	m.callback = nil
	return nil
}

func (m *darwinManager) Close() error {
	C.unregisterHotkey()
	m.callback = nil
	globalManager = nil
	return nil
}