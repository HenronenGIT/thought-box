package com.thoughtbox.http

import com.thoughtbox.config.LimitsConfig
import com.thoughtbox.domain.Thought
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// DTOs are wire-format objects. Node.js mental model: response serializers.
// Keeping them separate from domain models lets internal models evolve safely.
@Serializable
data class HealthResponse(val ok: Boolean = true, val env: String)

@Serializable
data class ConfigResponse(
    @SerialName("max_duration_ms") val maxDurationMs: Long,
    @SerialName("min_duration_ms") val minDurationMs: Long,
    @SerialName("max_size_bytes") val maxSizeBytes: Long,
) {
    companion object {
        fun from(config: LimitsConfig) = ConfigResponse(config.maxDurationMs, config.minDurationMs, config.maxSizeBytes)
    }
}

@Serializable
data class MeResponse(@SerialName("user_id") val userId: String)

@Serializable
data class CreateThoughtResponse(val id: String, val status: String, @SerialName("created_at") val createdAt: String)

@Serializable
data class ThoughtResponse(
    val id: String,
    val status: String,
    @SerialName("created_at") val createdAt: String,
    @SerialName("updated_at") val updatedAt: String,
    val transcript: String?,
    val audio: AudioResponse,
    val enrichment: EnrichmentResponse?,
    @SerialName("last_error") val lastError: String?,
) {
    companion object {
        fun from(thought: Thought) = ThoughtResponse(
            id = thought.id.toString(),
            status = thought.status.wireValue(),
            createdAt = thought.createdAt.toString(),
            updatedAt = thought.updatedAt.toString(),
            transcript = thought.transcript,
            audio = AudioResponse(thought.mimeType, thought.durationMs, thought.sizeBytes),
            enrichment = thought.enrichment?.let {
                EnrichmentResponse(it.category.name, it.tags, it.title, it.summary, it.model, it.promptVersion)
            },
            lastError = thought.lastError,
        )
    }
}

@Serializable
data class AudioResponse(
    @SerialName("mime_type") val mimeType: String,
    @SerialName("duration_ms") val durationMs: Long?,
    @SerialName("size_bytes") val sizeBytes: Long,
)

@Serializable
data class EnrichmentResponse(
    val category: String,
    val tags: List<String>,
    val title: String,
    val summary: String,
    val model: String,
    @SerialName("prompt_version") val promptVersion: String,
)

@Serializable
data class ThoughtListResponse(
    val items: List<ThoughtResponse>,
    @SerialName("next_cursor") val nextCursor: String?,
)

@Serializable
data class ErrorResponse(val error: String)
