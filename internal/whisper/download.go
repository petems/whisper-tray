package whisper

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Model download URLs (Hugging Face)
var modelURLs = map[string]string{
	"base.en":   "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin",
	"small.en":  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.en.bin",
	"medium.en": "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.en.bin",
	"large-v3":  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin",
}

// downloadModel downloads a Whisper model if it doesn't exist
func downloadModel(model string, destPath string) error {
	url, ok := modelURLs[model]
	if !ok {
		return fmt.Errorf("unknown model: %s", model)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create models directory: %w", err)
	}

	// Download to temp file first
	tmpPath := destPath + ".tmp"
	defer os.Remove(tmpPath)

	fmt.Printf("Downloading model %s...\n", model)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download model: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	// TODO: Add progress tracking
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}

	// Move to final location
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to move model file: %w", err)
	}

	fmt.Printf("Model %s downloaded successfully\n", model)
	return nil
}

// TODO: Add SHA256 verification
// TODO: Add resumable downloads
// TODO: Add progress bar