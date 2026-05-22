package validation

import (
	"testing"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/config"
)

func TestAudioValidation(t *testing.T) {
	limits := config.Limits{MinDurationMs: 1000, MaxDurationMs: 60000, MaxSizeBytes: 10}
	if err := ValidateMimeType("audio/webm"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDuration(999, limits); err == nil {
		t.Fatal("expected short duration error")
	}
	if err := ValidateSize(11, limits); err == nil {
		t.Fatal("expected size error")
	}
}
