package com.thoughtbox.domain

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import java.time.Instant
import java.util.UUID

// Stable single-user id for v1. Future auth swaps UserResolver, not the schema.
val SeededUserId: UUID = UUID.fromString("00000000-0000-4000-8000-000000000001")

// Kotlin enum = closed set of values. Node.js mental model: a string union, but
// checked at compile time and exhaustively handled in when expressions.
enum class Status {
    @SerialName("pending")
    Pending,
    @SerialName("transcribing")
    Transcribing,
    @SerialName("enriching")
    Enriching,
    @SerialName("done")
    Done,
    @SerialName("failed_transcription")
    FailedTranscription,
    @SerialName("failed_enrichment")
    FailedEnrichment;

    fun wireValue(): String = name.replace(Regex("([a-z])([A-Z])"), "$1_$2").lowercase()

    companion object {
        fun fromDb(value: String): Status = entries.first { it.wireValue() == value }
    }
}

// Events are commands that move a Thought through the pipeline state machine.
enum class StatusEvent {
    StartTranscription,
    TranscriptionSucceeded,
    TranscriptionFailedTerminal,
    EnrichmentSucceeded,
    EnrichmentFailedTerminal,
    Recover,
}

// Pure transition function. Illegal transitions fail fast instead of silently
// writing nonsense status values to the database.
fun Status.next(event: StatusEvent): Status = when (this to event) {
    Status.Pending to StatusEvent.StartTranscription -> Status.Transcribing
    Status.Transcribing to StatusEvent.TranscriptionSucceeded -> Status.Enriching
    Status.Transcribing to StatusEvent.TranscriptionFailedTerminal -> Status.FailedTranscription
    Status.Enriching to StatusEvent.EnrichmentSucceeded -> Status.Done
    Status.Enriching to StatusEvent.EnrichmentFailedTerminal -> Status.FailedEnrichment
    Status.Transcribing to StatusEvent.Recover -> Status.Pending
    Status.Enriching to StatusEvent.Recover -> Status.Enriching
    else -> throw IllegalStateException("Illegal transition: $this + $event")
}

// Closed category set used by Kotlin code, DB check constraint, and frontend chips.
enum class Category {
    idea,
    todo,
    feeling,
    question,
    observation,
    reminder,
}

// Domain model returned by the repository. It is deliberately not the HTTP JSON
// shape; DTOs in http/ decide what is exposed over the wire.
data class Thought(
    val id: UUID,
    val userId: UUID,
    val createdAt: Instant,
    val updatedAt: Instant,
    val audioS3Key: String,
    val mimeType: String,
    val durationMs: Long?,
    val sizeBytes: Long,
    val transcript: String?,
    val status: Status,
    val attempts: Int,
    val lastError: String?,
    val transcribedAt: Instant?,
    val enrichment: ThoughtEnrichment?,
)

data class ThoughtEnrichment(
    val category: Category,
    val tags: List<String>,
    val title: String,
    val summary: String,
    val model: String,
    val promptVersion: String,
)

data class AudioBlob(
    val key: String,
    val mimeType: String,
    val sizeBytes: Long,
)

data class StoredAudio(
    val bytes: java.io.InputStream,
    val contentType: String,
    val contentLength: Long?,
)
