package domain

import "testing"

func TestDefaultModeFor(t *testing.T) {
	cases := []struct {
		category Category
		want     EchoMode
	}{
		{CategoryFeeling, EchoModeMirror},
		{CategoryIdea, EchoModeChallenger},
		{CategoryObservation, EchoModeReframer},
		{CategoryLearning, EchoModeExtender},
	}
	for _, c := range cases {
		got, ok := DefaultModeFor(c.category)
		if !ok {
			t.Fatalf("category %s: expected ok=true", c.category)
		}
		if got != c.want {
			t.Fatalf("category %s: got %s, want %s", c.category, got, c.want)
		}
	}

	if _, ok := DefaultModeFor(Category("unknown")); ok {
		t.Fatal("expected unknown category to return ok=false")
	}
}

func TestParseEchoMode(t *testing.T) {
	for _, mode := range []string{"mirror", "challenger", "reframer", "extender"} {
		parsed, ok := ParseEchoMode(mode)
		if !ok || string(parsed) != mode {
			t.Fatalf("expected %s to parse", mode)
		}
	}
	if _, ok := ParseEchoMode("garbage"); ok {
		t.Fatal("expected unknown mode to fail")
	}
}
