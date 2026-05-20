package com.thoughtbox.validation

import com.thoughtbox.config.LimitsConfig
import java.time.Instant

// Browser recorder MIME types the backend accepts. Client validation is UX;
// server validation is the real trust boundary.
val SupportedMimeTypes = setOf("audio/webm", "audio/webm;codecs=opus", "audio/mp4", "audio/mpeg", "audio/wav")

// Small pure helpers. Node.js mental model: zod/joi-like validation, but plain
// Kotlin functions that throw IllegalArgumentException on bad input.
fun validateMimeType(mimeType: String) {
    require(mimeType in SupportedMimeTypes) { "Unsupported audio MIME type: $mimeType" }
}

fun validateDuration(durationMs: Long, limits: LimitsConfig) {
    require(durationMs in limits.minDurationMs..limits.maxDurationMs) {
        "Duration must be between ${limits.minDurationMs} and ${limits.maxDurationMs} ms"
    }
}

fun validateSize(sizeBytes: Long, limits: LimitsConfig) {
    require(sizeBytes <= limits.maxSizeBytes) { "Audio exceeds max size ${limits.maxSizeBytes}" }
}

fun parseCursor(value: String?): Instant? = value?.let {
    runCatching { Instant.parse(it) }.getOrElse { throw IllegalArgumentException("Malformed cursor") }
}

fun parseLimit(value: String?): Int = value?.toIntOrNull()?.coerceIn(1, 100) ?: 50
