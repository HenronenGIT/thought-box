package pipeline

import "time"

type RetryDecision struct {
	Retry bool
	Delay time.Duration
}

func retryDecision(attemptsSoFar int) RetryDecision {
	switch attemptsSoFar {
	case 0:
		return RetryDecision{Retry: true, Delay: time.Second}
	case 1:
		return RetryDecision{Retry: true, Delay: 4 * time.Second}
	case 2:
		return RetryDecision{Retry: true, Delay: 16 * time.Second}
	default:
		return RetryDecision{}
	}
}
