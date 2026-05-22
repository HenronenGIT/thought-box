package pipeline

import (
	"testing"
	"time"
)

func TestRetryDecision(t *testing.T) {
	if decision := retryDecision(0); !decision.Retry || decision.Delay != time.Second {
		t.Fatalf("unexpected first retry: %#v", decision)
	}
	if decision := retryDecision(3); decision.Retry {
		t.Fatalf("expected terminal failure: %#v", decision)
	}
}
