package audio

import "testing"

func TestDownmixInterleavedMono(t *testing.T) {
	input := []float32{0.1, 0.2, 0.3, 0.4}
	got := downmixInterleaved(input, 1, len(input))

	if len(got) != len(input) {
		t.Fatalf("expected %d samples, got %d", len(input), len(got))
	}
	for i := range input {
		if got[i] != input[i] {
			t.Fatalf("expected element %d to be %f, got %f", i, input[i], got[i])
		}
	}

	if &got[0] == &input[0] {
		t.Fatal("expected mono result to be copied into a new slice")
	}
}

func TestDownmixInterleavedStereo(t *testing.T) {
	frames := 4
	input := []float32{
		0.0, 1.0,
		0.5, 0.5,
		1.0, 0.0,
		-0.5, 0.5,
	}

	expected := []float32{
		0.5, 0.5, 0.5, 0.0,
	}

	got := downmixInterleaved(input, 2, frames)
	if len(got) != len(expected) {
		t.Fatalf("expected %d frames, got %d", len(expected), len(got))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("frame %d mismatch: expected %f, got %f", i, expected[i], got[i])
		}
	}
}

func TestDownmixInterleavedMoreChannels(t *testing.T) {
	frames := 2
	input := []float32{
		1, 3, 5,
		2, 4, 6,
	}

	expected := []float32{3, 4}

	got := downmixInterleaved(input, 3, frames)
	for i := range expected {
		if got[i] != expected[i] {
			t.Fatalf("frame %d mismatch: expected %f, got %f", i, expected[i], got[i])
		}
	}
}
