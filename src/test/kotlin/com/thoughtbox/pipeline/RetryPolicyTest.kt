package com.thoughtbox.pipeline

import io.kotest.matchers.shouldBe
import org.junit.jupiter.api.Test
import kotlin.time.Duration.Companion.seconds

class RetryPolicyTest {
    @Test
    fun `retries three times then fails`() {
        retryDecision(0) shouldBe RetryDecision.RetryAfter(1.seconds)
        retryDecision(1) shouldBe RetryDecision.RetryAfter(4.seconds)
        retryDecision(2) shouldBe RetryDecision.RetryAfter(16.seconds)
        retryDecision(3) shouldBe RetryDecision.Fail
    }
}

