package com.thoughtbox.validation

import com.thoughtbox.config.LimitsConfig
import io.kotest.assertions.throwables.shouldThrow
import org.junit.jupiter.api.Test

class ValidatorsTest {
    private val limits = LimitsConfig(maxDurationMs = 60_000, minDurationMs = 1_000, maxSizeBytes = 10)

    @Test
    fun `accepts supported mime and valid duration`() {
        validateMimeType("audio/webm;codecs=opus")
        validateDuration(1_000, limits)
        validateDuration(60_000, limits)
        validateSize(10, limits)
    }

    @Test
    fun `rejects invalid values`() {
        shouldThrow<IllegalArgumentException> { validateMimeType("text/plain") }
        shouldThrow<IllegalArgumentException> { validateDuration(999, limits) }
        shouldThrow<IllegalArgumentException> { validateDuration(60_001, limits) }
        shouldThrow<IllegalArgumentException> { validateSize(11, limits) }
    }
}

