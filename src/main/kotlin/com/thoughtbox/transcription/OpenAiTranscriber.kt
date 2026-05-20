package com.thoughtbox.transcription

import com.thoughtbox.domain.AudioBlob
import com.thoughtbox.storage.BlobStore
import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.engine.cio.CIO
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.forms.formData
import io.ktor.client.request.forms.submitFormWithBinaryData
import io.ktor.client.request.header
import io.ktor.client.statement.HttpResponse
import io.ktor.http.Headers
import io.ktor.http.HttpHeaders
import io.ktor.http.isSuccess
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import org.slf4j.LoggerFactory
import io.ktor.utils.io.streams.asInput
import kotlin.system.measureTimeMillis

// Hand-rolled Whisper client using Ktor HttpClient. It depends on BlobStore so
// it can fetch audio by key, then send multipart/form-data to OpenAI.
class OpenAiTranscriber(
    private val apiKey: String,
    private val blobStore: BlobStore,
    private val client: HttpClient = HttpClient(CIO) {
        install(ContentNegotiation) { json(Json { ignoreUnknownKeys = true }) }
    },
) : Transcriber {
    private val logger = LoggerFactory.getLogger(javaClass)

    // suspend means this can pause while doing I/O without blocking an OS thread.
    // Node.js mental model: an async function that awaits HTTP/network work.
    override suspend fun transcribe(audio: AudioBlob): TranscriptionResult {
        val stored = blobStore.get(audio.key)
        lateinit var response: HttpResponse
        var status = 0
        val duration = measureTimeMillis {
            response = client.submitFormWithBinaryData(
                url = "https://api.openai.com/v1/audio/transcriptions",
                formData = formData {
                    append("model", "whisper-1")
                    appendInput(
                        "file",
                        Headers.build {
                            append(HttpHeaders.ContentType, audio.mimeType)
                            append(HttpHeaders.ContentDisposition, "filename=\"thought\"")
                        },
                        stored.contentLength ?: audio.sizeBytes,
                    ) { stored.bytes.asInput() }
                },
            ) {
                header(HttpHeaders.Authorization, "Bearer $apiKey")
            }
            status = response.status.value
        }
        logger.info("external_api provider=openai-whisper duration_ms={} response_status={}", duration, status)
        if (!response.status.isSuccess()) error("Whisper failed with status ${response.status.value}")
        return TranscriptionResult(response.body<WhisperResponse>().text, "whisper-1")
    }
}

@Serializable
private data class WhisperResponse(
    @SerialName("text") val text: String,
)
