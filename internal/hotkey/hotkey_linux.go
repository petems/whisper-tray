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

// New creates a new Linux hotkey manager using X11
func New() (Manager, error) {
	mgr := &linuxManager{
		callbacks: make(map[int]func(bool)),
		stop:      make(chan struct{}),
	}

	go mgr.eventLoop()

	return mgr, nil
}

func (m *linuxManager) Register(accel string, callback func(pressed bool)) error {
	// TODO: Parse "Alt+Space" -> keycode and modifiers
	// For now, hardcoded to Space (keycode 65) + Alt (Mod1Mask = 8)
	keycode := 65      // Space
	modifiers := 8     // Mod1Mask (Alt)

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
	// TODO: XUngrabKey implementation
	return nil
}

func (m *linuxManager) Close() error {
	close(m.stop)
	return nil
}