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
	"strings"
)

const (
	cmdKeyMask     = uint32(1 << 8)
	shiftKeyMask   = uint32(1 << 9)
	optionKeyMask  = uint32(1 << 11)
	controlKeyMask = uint32(1 << 12)
)

var modifierLookup = map[string]uint32{
	"cmd":     cmdKeyMask,
	"command": cmdKeyMask,
	"meta":    cmdKeyMask,
	"super":   cmdKeyMask,
	"shift":   shiftKeyMask,
	"option":  optionKeyMask,
	"opt":     optionKeyMask,
	"alt":     optionKeyMask,
	"ctrl":    controlKeyMask,
	"control": controlKeyMask,
}

var keyLookup = map[string]uint32{
	"SPACE":     uint32(C.kVK_Space),
	"TAB":       uint32(C.kVK_Tab),
	"ESC":       uint32(C.kVK_Escape),
	"ESCAPE":    uint32(C.kVK_Escape),
	"RETURN":    uint32(C.kVK_Return),
	"ENTER":     uint32(C.kVK_Return),
	"DELETE":    uint32(C.kVK_Delete),
	"BACKSPACE": uint32(C.kVK_Delete),
	"GRAVE":     uint32(C.kVK_ANSI_Grave),
	"BACKQUOTE": uint32(C.kVK_ANSI_Grave),
}

func init() {
	letterMap := map[string]uint32{
		"A": uint32(C.kVK_ANSI_A),
		"B": uint32(C.kVK_ANSI_B),
		"C": uint32(C.kVK_ANSI_C),
		"D": uint32(C.kVK_ANSI_D),
		"E": uint32(C.kVK_ANSI_E),
		"F": uint32(C.kVK_ANSI_F),
		"G": uint32(C.kVK_ANSI_G),
		"H": uint32(C.kVK_ANSI_H),
		"I": uint32(C.kVK_ANSI_I),
		"J": uint32(C.kVK_ANSI_J),
		"K": uint32(C.kVK_ANSI_K),
		"L": uint32(C.kVK_ANSI_L),
		"M": uint32(C.kVK_ANSI_M),
		"N": uint32(C.kVK_ANSI_N),
		"O": uint32(C.kVK_ANSI_O),
		"P": uint32(C.kVK_ANSI_P),
		"Q": uint32(C.kVK_ANSI_Q),
		"R": uint32(C.kVK_ANSI_R),
		"S": uint32(C.kVK_ANSI_S),
		"T": uint32(C.kVK_ANSI_T),
		"U": uint32(C.kVK_ANSI_U),
		"V": uint32(C.kVK_ANSI_V),
		"W": uint32(C.kVK_ANSI_W),
		"X": uint32(C.kVK_ANSI_X),
		"Y": uint32(C.kVK_ANSI_Y),
		"Z": uint32(C.kVK_ANSI_Z),
	}

	for k, v := range letterMap {
		keyLookup[k] = v
	}

	digitMap := map[string]uint32{
		"0": uint32(C.kVK_ANSI_0),
		"1": uint32(C.kVK_ANSI_1),
		"2": uint32(C.kVK_ANSI_2),
		"3": uint32(C.kVK_ANSI_3),
		"4": uint32(C.kVK_ANSI_4),
		"5": uint32(C.kVK_ANSI_5),
		"6": uint32(C.kVK_ANSI_6),
		"7": uint32(C.kVK_ANSI_7),
		"8": uint32(C.kVK_ANSI_8),
		"9": uint32(C.kVK_ANSI_9),
	}

	for k, v := range digitMap {
		keyLookup[k] = v
	}

	functionKeyMap := map[string]uint32{
		"F1":  uint32(C.kVK_F1),
		"F2":  uint32(C.kVK_F2),
		"F3":  uint32(C.kVK_F3),
		"F4":  uint32(C.kVK_F4),
		"F5":  uint32(C.kVK_F5),
		"F6":  uint32(C.kVK_F6),
		"F7":  uint32(C.kVK_F7),
		"F8":  uint32(C.kVK_F8),
		"F9":  uint32(C.kVK_F9),
		"F10": uint32(C.kVK_F10),
		"F11": uint32(C.kVK_F11),
		"F12": uint32(C.kVK_F12),
	}

	for k, v := range functionKeyMap {
		keyLookup[k] = v
	}
}

func parseAccelerator(accel string) (C.UInt32, C.UInt32, error) {
	if accel == "" {
		return 0, 0, fmt.Errorf("accelerator string is empty")
	}

	tokens := strings.Split(accel, "+")
	var modifiers uint32
	var keyToken string

	for _, token := range tokens {
		t := strings.TrimSpace(token)
		if t == "" {
			continue
		}

		lower := strings.ToLower(t)
		if mask, ok := modifierLookup[lower]; ok {
			modifiers |= mask
			continue
		}

		keyToken = strings.ToUpper(t)
	}

	if keyToken == "" {
		return 0, 0, fmt.Errorf("missing base key in accelerator %q", accel)
	}

	keyCode, ok := keyLookup[keyToken]
	if !ok {
		return 0, 0, fmt.Errorf("unsupported key %q", keyToken)
	}

	return C.UInt32(keyCode), C.UInt32(modifiers), nil
}

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

	keyCode, modifiers, err := parseAccelerator(accel)
	if err != nil {
		return fmt.Errorf("failed to parse accelerator %q: %w", accel, err)
	}

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
