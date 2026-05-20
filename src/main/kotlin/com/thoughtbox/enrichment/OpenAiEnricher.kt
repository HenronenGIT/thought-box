package com.thoughtbox.enrichment

import com.thoughtbox.domain.Category
import com.thoughtbox.domain.ThoughtEnrichment
import io.ktor.client.HttpClient
import io.ktor.client.call.body
import io.ktor.client.engine.cio.CIO
import io.ktor.client.plugins.contentnegotiation.ContentNegotiation
import io.ktor.client.request.header
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.http.ContentType
import io.ktor.http.HttpHeaders
import io.ktor.http.contentType
import io.ktor.http.isSuccess
import io.ktor.serialization.kotlinx.json.json
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.add
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put
import kotlinx.serialization.json.putJsonArray
import kotlinx.serialization.json.putJsonObject
import org.slf4j.LoggerFactory
import java.util.UUID
import kotlin.system.measureTimeMillis

private const val PromptVersion = "v1"
private const val Model = "gpt-4o-mini"

// Hand-rolled OpenAI chat client. It asks for structured JSON so the rest of the
// app can parse a normal Kotlin data class instead of prompt-shaped text.
class OpenAiEnricher(
    private val apiKey: String,
    private val client: HttpClient = HttpClient(CIO) {
        install(ContentNegotiation) { json(Json { ignoreUnknownKeys = true }) }
    },
) : Enricher {
    private val logger = LoggerFactory.getLogger(javaClass)
    private val json = Json { ignoreUnknownKeys = true }

    // thoughtId participates in Idempotency-Key so a retry should not double-charge
    // or produce conflicting enrichment for the same prompt version.
    override suspend fun enrich(thoughtId: UUID, transcript: String): ThoughtEnrichment {
        lateinit var response: HttpResponse
        var status = 0
        val duration = measureTimeMillis {
            response = client.post("https://api.openai.com/v1/chat/completions") {
                header(HttpHeaders.Authorization, "Bearer $apiKey")
                header("Idempotency-Key", "$thoughtId:$PromptVersion")
                contentType(ContentType.Application.Json)
                setBody(requestBody(transcript))
            }
            status = response.status.value
        }
        logger.info("external_api provider=openai-chat duration_ms={} response_status={}", duration, status)
        if (!response.status.isSuccess()) error("Chat completion failed with status ${response.status.value}")
        val completion = response.body<ChatCompletionResponse>()
        val content = completion.choices.firstOrNull()?.message?.content ?: error("Missing enrichment content")
        val parsed = json.decodeFromString(StructuredEnrichment.serializer(), content)
        return ThoughtEnrichment(
            category = Category.valueOf(parsed.category),
            tags = parsed.tags.map { it.trim().lowercase() }.filter { it.isNotBlank() }.distinct().take(8),
            title = parsed.title,
            summary = parsed.summary,
            model = Model,
            promptVersion = PromptVersion,
        )
    }

    // Builds the Chat Completions JSON body with response_format=json_schema.
    private fun requestBody(transcript: String) = buildJsonObject {
        put("model", Model)
        putJsonArray("messages") {
            add(buildJsonObject {
                put("role", "system")
                put("content", "Return concise JSON for a dictated thought. Categories: idea,todo,feeling,question,observation,reminder.")
            })
            add(buildJsonObject {
                put("role", "user")
                put("content", transcript)
            })
        }
        putJsonObject("response_format") {
            put("type", "json_schema")
            putJsonObject("json_schema") {
                put("name", "thought_enrichment")
                put("strict", true)
                putJsonObject("schema") {
                    put("type", "object")
                    putJsonObject("properties") {
                        putJsonObject("category") {
                            put("type", "string")
                            putJsonArray("enum") { Category.entries.forEach { add(it.name) } }
                        }
                        putJsonObject("tags") {
                            put("type", "array")
                            putJsonObject("items") { put("type", "string") }
                        }
                        putJsonObject("title") { put("type", "string") }
                        putJsonObject("summary") { put("type", "string") }
                    }
                    putJsonArray("required") {
                        add("category")
                        add("tags")
                        add("title")
                        add("summary")
                    }
                    put("additionalProperties", false)
                }
            }
        }
    }
}

// This is the JSON shape the model must return inside message.content.
@Serializable
data class StructuredEnrichment(
    val category: String,
    val tags: List<String>,
    val title: String,
    val summary: String,
)

@Serializable
private data class ChatCompletionResponse(val choices: List<Choice>)

@Serializable
private data class Choice(val message: Message)

@Serializable
private data class Message(
    val content: String? = null,
    val refusal: String? = null,
)
