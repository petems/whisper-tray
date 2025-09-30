package whisper

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
)

// Model download URLs (Hugging Face)
var modelURLs = map[string]string{
	"base.en":         "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin",
	"small.en":        "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.en.bin",
	"medium.en":       "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.en.bin",
	"large-v3":        "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin",
	"large-v3-turbo":  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin",
}

// progressWriter wraps an io.Writer to track download progress
type progressWriter struct {
	total      int64
	downloaded int64
	lastLog    time.Time
	model      string
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.downloaded += int64(n)

	// Log progress every 2 seconds or when complete
	now := time.Now()
	if now.Sub(pw.lastLog) >= 2*time.Second || pw.downloaded >= pw.total {
		pw.lastLog = now
		percent := float64(pw.downloaded) / float64(pw.total) * 100
		mbDownloaded := float64(pw.downloaded) / 1024 / 1024
		mbTotal := float64(pw.total) / 1024 / 1024

		log.Info().
			Str("model", pw.model).
			Float64("percent", percent).
			Float64("downloaded_mb", mbDownloaded).
			Float64("total_mb", mbTotal).
			Msg("Downloading model")
	}

	return n, nil
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

	log.Info().Str("model", model).Str("url", url).Msg("Starting model download")

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download model: HTTP %d", resp.StatusCode)
	}

	// Get content length for progress tracking
	totalSize := resp.ContentLength
	if totalSize <= 0 {
		log.Warn().Str("model", model).Msg("Content-Length not provided, progress tracking unavailable")
	}

	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer out.Close()

	// Use progress writer if we know the size
	var writer io.Writer = out
	if totalSize > 0 {
		pw := &progressWriter{
			total:   totalSize,
			model:   model,
			lastLog: time.Now(),
		}
		// Use MultiWriter to write to both file and progress tracker
		writer = io.MultiWriter(out, pw)
	}

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write model file: %w", err)
	}

	// Move to final location
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to move model file: %w", err)
	}

	log.Info().
		Str("model", model).
		Str("path", destPath).
		Float64("size_mb", float64(totalSize)/1024/1024).
		Msg("Model downloaded successfully")

	return nil
}

// TODO: Add SHA256 verification
// TODO: Add resumable downloads