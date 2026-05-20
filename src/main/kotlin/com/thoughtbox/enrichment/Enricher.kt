package com.thoughtbox.enrichment

import com.thoughtbox.domain.ThoughtEnrichment
import java.util.UUID

// Provider boundary for transcript enrichment. Node.js mental model: an
// interface you would fake in tests and implement with one OpenAI client module.
interface Enricher {
    suspend fun enrich(thoughtId: UUID, transcript: String): ThoughtEnrichment
}
