package echo

import (
	"strings"
	"testing"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/domain"
)

func TestAllModesHaveLoadBearingConstraints(t *testing.T) {
	modes := []domain.EchoMode{
		domain.EchoModeMirror,
		domain.EchoModeChallenger,
		domain.EchoModeReframer,
		domain.EchoModeExtender,
	}
	for _, mode := range modes {
		prompt, ok := PromptFor(mode)
		if !ok {
			t.Fatalf("%s prompt missing", mode)
		}
		mustContain(t, string(mode), prompt, []string{
			"English",
			"40 words",
			"second person",
			"No advice",
			"No validation",
			"No emoji",
			"empty string",
		})
	}
}

func TestModeSpecificFlavor(t *testing.T) {
	cases := map[domain.EchoMode]string{
		domain.EchoModeMirror:     "Mirror",
		domain.EchoModeChallenger: "Challenger",
		domain.EchoModeReframer:   "Reframer",
		domain.EchoModeExtender:   "Extender",
	}
	for mode, fragment := range cases {
		prompt, _ := PromptFor(mode)
		mustContain(t, string(mode), prompt, []string{fragment})
	}
}

func mustContain(t *testing.T, name, body string, fragments []string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(body, fragment) {
			t.Errorf("%s prompt missing fragment %q", name, fragment)
		}
	}
}
