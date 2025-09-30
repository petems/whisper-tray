package app

import (
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/petems/whisper-tray/internal/config"
	"github.com/rs/zerolog"
)

type fakeSession struct {
	partials       chan string
	finals         chan string
	partialsCalled chan struct{}
	finalsCalled   chan struct{}

	partialsOnce sync.Once
	finalsOnce   sync.Once
}

func newFakeSession() *fakeSession {
	return &fakeSession{
		partials:       make(chan string, 4),
		finals:         make(chan string, 4),
		partialsCalled: make(chan struct{}),
		finalsCalled:   make(chan struct{}),
	}
}

func (s *fakeSession) Feed(_ []float32) error { return nil }

func (s *fakeSession) Partials() <-chan string {
	s.partialsOnce.Do(func() { close(s.partialsCalled) })
	return s.partials
}

func (s *fakeSession) Finals() <-chan string {
	s.finalsOnce.Do(func() { close(s.finalsCalled) })
	return s.finals
}

func (s *fakeSession) Close() error { return nil }

func startCollector(t *testing.T, a *App, done chan struct{}) <-chan error {
	t.Helper()
	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errCh <- asError(r)
				return
			}
			errCh <- nil
		}()
		a.collectTranscripts(done)
	}()

	return errCh
}

func asError(r interface{}) error {
	if err, ok := r.(error); ok {
		return err
	}
	return fmt.Errorf("panic: %v", r)
}

func waitForSignal(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	if ch == nil {
		return
	}
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s", msg)
	}
}

func waitForClosed(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for %s", msg)
	}
}

func TestCollectTranscriptsBuffersFinals(t *testing.T) {
	app := &App{
		cfg: &config.Config{},
		log: zerolog.New(io.Discard),
	}

	session := newFakeSession()
	app.session = session

	done := make(chan struct{})
	app.collectorDone = done
	errCh := startCollector(t, app, done)

	waitForSignal(t, session.finalsCalled, "collector to read finals channel")

	session.partials <- "partial ignored"
	session.finals <- "final transcript"
	close(session.finals)

	waitForClosed(t, done, "collector completion")

	if err := <-errCh; err != nil {
		t.Fatalf("collector returned error: %v", err)
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	if len(app.textBuffer) != 1 || app.textBuffer[0] != "final transcript" {
		t.Fatalf("expected buffered final, got %#v", app.textBuffer)
	}
}

func TestCollectTranscriptsReassignmentSafe(t *testing.T) {
	app := &App{
		cfg: &config.Config{},
		log: zerolog.New(io.Discard),
	}

	// Start first collector
	session1 := newFakeSession()
	done1 := make(chan struct{})
	app.session = session1
	app.collectorDone = done1
	errCh1 := startCollector(t, app, done1)
	waitForSignal(t, session1.finalsCalled, "first collector to read finals channel")

	// Start second collector before the first one finishes to simulate a restart
	session2 := newFakeSession()
	done2 := make(chan struct{})
	app.session = session2
	app.collectorDone = done2
	errCh2 := startCollector(t, app, done2)
	waitForSignal(t, session2.finalsCalled, "second collector to read finals channel")

	// Finish the first collector and ensure it closes its original done channel.
	close(session1.finals)
	waitForClosed(t, done1, "first collector to finish")

	select {
	case <-done2:
		t.Fatalf("second collector done channel closed unexpectedly")
	default:
	}

	// Send a final transcript through the second collector and close it.
	session2.finals <- "second final"
	close(session2.finals)
	waitForClosed(t, done2, "second collector to finish")

	for i, errCh := range []<-chan error{errCh1, errCh2} {
		if err := <-errCh; err != nil {
			t.Fatalf("collector %d returned error: %v", i+1, err)
		}
	}

	app.mu.Lock()
	defer app.mu.Unlock()

	if len(app.textBuffer) != 1 || app.textBuffer[0] != "second final" {
		t.Fatalf("expected only the second final buffered, got %#v", app.textBuffer)
	}
}
