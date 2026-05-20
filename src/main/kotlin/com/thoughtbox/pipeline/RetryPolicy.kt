package com.thoughtbox.pipeline

import kotlin.time.Duration
import kotlin.time.Duration.Companion.seconds

// Sealed interface = closed result union. Node.js mental model: a discriminated
// union like { type: "retry", delayMs } | { type: "fail" }.
sealed interface RetryDecision {
    data class RetryAfter(val delay: Duration) : RetryDecision
    data object Fail : RetryDecision
}

// attemptsSoFar is the count before recording the current failure.
fun retryDecision(attemptsSoFar: Int): RetryDecision = when (attemptsSoFar) {
    0 -> RetryDecision.RetryAfter(1.seconds)
    1 -> RetryDecision.RetryAfter(4.seconds)
    2 -> RetryDecision.RetryAfter(16.seconds)
    else -> RetryDecision.Fail
}
