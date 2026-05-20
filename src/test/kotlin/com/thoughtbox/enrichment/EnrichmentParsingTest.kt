package com.thoughtbox.enrichment

import io.kotest.assertions.throwables.shouldThrow
import io.kotest.matchers.shouldBe
import kotlinx.serialization.json.Json
import org.junit.jupiter.api.Test

class EnrichmentParsingTest {
    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun `parses structured enrichment`() {
        val parsed = json.decodeFromString(
            StructuredEnrichment.serializer(),
            """{"category":"idea","tags":["kotlin"],"title":"Learn Kotlin","summary":"A note about learning Kotlin."}""",
        )

        parsed.category shouldBe "idea"
        parsed.tags shouldBe listOf("kotlin")
    }

    @Test
    fun `rejects malformed enrichment`() {
        shouldThrow<Exception> {
            json.decodeFromString(StructuredEnrichment.serializer(), """{"refusal":"no"}""")
        }
    }
}

