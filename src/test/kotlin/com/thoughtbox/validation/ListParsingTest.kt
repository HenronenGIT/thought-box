package com.thoughtbox.validation

import io.kotest.assertions.throwables.shouldThrow
import io.kotest.matchers.shouldBe
import org.junit.jupiter.api.Test
import java.time.Instant

class ListParsingTest {
    @Test
    fun `parses cursors and limits`() {
        parseCursor("2026-05-20T10:00:00Z") shouldBe Instant.parse("2026-05-20T10:00:00Z")
        parseLimit(null) shouldBe 50
        parseLimit("200") shouldBe 100
    }

    @Test
    fun `rejects bad cursor`() {
        shouldThrow<IllegalArgumentException> { parseCursor("nope") }
    }
}

