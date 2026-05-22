package validation

import (
	"fmt"
	"slices"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/config"
)

var SupportedMimeTypes = []string{"audio/webm", "audio/webm;codecs=opus", "audio/mp4", "audio/mpeg", "audio/wav"}

func ValidateMimeType(mimeType string) error {
	if !slices.Contains(SupportedMimeTypes, mimeType) {
		return fmt.Errorf("Unsupported audio MIME type: %s", mimeType)
	}
	return nil
}

func ValidateDuration(durationMs int64, limits config.Limits) error {
	if durationMs < limits.MinDurationMs || durationMs > limits.MaxDurationMs {
		return fmt.Errorf("Duration must be between %d and %d ms", limits.MinDurationMs, limits.MaxDurationMs)
	}
	return nil
}

func ValidateSize(sizeBytes int64, limits config.Limits) error {
	if sizeBytes > limits.MaxSizeBytes {
		return fmt.Errorf("Audio exceeds max size %d", limits.MaxSizeBytes)
	}
	return nil
}
