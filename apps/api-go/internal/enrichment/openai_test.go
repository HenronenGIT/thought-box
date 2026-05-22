package enrichment

import "testing"

func TestNormalizeTags(t *testing.T) {
	tags := normalizeTags([]string{" AI ", "ai", "", "Go"})
	if len(tags) != 2 || tags[0] != "ai" || tags[1] != "go" {
		t.Fatalf("unexpected tags: %#v", tags)
	}
}
