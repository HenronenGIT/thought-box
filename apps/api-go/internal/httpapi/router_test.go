package httpapi

import "testing"

func TestParseLimit(t *testing.T) {
	limit, err := parseLimit("")
	if err != nil || limit != 20 {
		t.Fatalf("default limit = %d, %v", limit, err)
	}
	if _, err := parseLimit("0"); err == nil {
		t.Fatal("expected invalid low limit")
	}
	if _, err := parseLimit("101"); err == nil {
		t.Fatal("expected invalid high limit")
	}
}
