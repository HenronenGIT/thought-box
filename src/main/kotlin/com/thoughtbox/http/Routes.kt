package com.thoughtbox.http

import com.thoughtbox.config.Config
import com.thoughtbox.db.ThoughtRepository
import com.thoughtbox.domain.Category
import com.thoughtbox.storage.BlobStore
import com.thoughtbox.user.UserResolver
import com.thoughtbox.validation.parseCursor
import com.thoughtbox.validation.parseLimit
import com.thoughtbox.validation.validateDuration
import com.thoughtbox.validation.validateMimeType
import com.thoughtbox.validation.validateSize
import io.ktor.http.ContentType
import io.ktor.http.HttpStatusCode
import io.ktor.http.content.PartData
import io.ktor.http.content.forEachPart
import io.ktor.http.content.streamProvider
import io.ktor.server.application.Application
import io.ktor.server.application.call
import io.ktor.server.request.receiveMultipart
import io.ktor.server.response.respond
import io.ktor.server.response.respondOutputStream
import io.ktor.server.routing.get
import io.ktor.server.routing.post
import io.ktor.server.routing.routing
import java.nio.file.Files
import java.util.UUID
import kotlin.io.path.inputStream
import kotlin.io.path.outputStream

// HTTP API surface. Ktor DSL mental model: Express routes, but typed Kotlin and
// coroutine-aware handlers. Keep this layer thin.
fun Application.configureRoutes(
    config: Config,
    repository: ThoughtRepository,
    blobStore: BlobStore,
    userResolver: UserResolver,
) {
    routing {
        get("/healthz") {
            call.respond(HealthResponse(env = config.appEnv))
        }

        get("/config") {
            call.respond(ConfigResponse.from(config.limits))
        }

        get("/me") {
            call.respond(MeResponse(userResolver.currentUserId(call).toString()))
        }

        post("/thoughts") {
            // Multipart parsing streams file parts to a temp file, then BlobStore
            // streams that file to S3/MinIO. The app avoids keeping audio in memory.
            val userId = userResolver.currentUserId(call)
            var mimeType: String? = null
            var durationMs: Long? = null
            var tempFile = Files.createTempFile("thought-", ".audio")

            call.receiveMultipart().forEachPart { part ->
                when (part) {
                    is PartData.FormItem -> when (part.name) {
                        "mime_type" -> mimeType = part.value
                        "duration_ms" -> durationMs = part.value.toLongOrNull()
                    }
                    is PartData.FileItem -> {
                        part.streamProvider().use { input ->
                            tempFile.outputStream().use { output -> input.copyTo(output) }
                        }
                    }
                    else -> Unit
                }
                part.dispose()
            }

            val resolvedMime = mimeType ?: throw IllegalArgumentException("mime_type is required")
            val resolvedDuration = durationMs ?: throw IllegalArgumentException("duration_ms is required")
            val sizeBytes = Files.size(tempFile)
            validateMimeType(resolvedMime)
            validateDuration(resolvedDuration, config.limits)
            validateSize(sizeBytes, config.limits)

            val id = UUID.randomUUID()
            val key = "${config.appEnv}/$userId/$id"
            tempFile.inputStream().use { input -> blobStore.put(key, resolvedMime, sizeBytes, input) }
            Files.deleteIfExists(tempFile)

            val thought = repository.insertThought(id, userId, key, resolvedMime, resolvedDuration, sizeBytes)
            call.respond(HttpStatusCode.Created, CreateThoughtResponse(thought.id.toString(), thought.status.wireValue(), thought.createdAt.toString()))
        }

        get("/thoughts") {
            // Pagination/filter parsing is pure logic in validation/, then SQL does
            // user scoping and AND-composed filters.
            val userId = userResolver.currentUserId(call)
            val limit = parseLimit(call.request.queryParameters["limit"])
            val before = parseCursor(call.request.queryParameters["before"])
            val category = call.request.queryParameters["category"]?.let { Category.valueOf(it) }
            val tag = call.request.queryParameters["tag"]
            val items = repository.listThoughts(userId, limit + 1, before, category, tag)
            val page = items.take(limit)
            val nextCursor = if (items.size > limit) page.lastOrNull()?.createdAt?.toString() else null
            call.respond(ThoughtListResponse(page.map(ThoughtResponse::from), nextCursor))
        }

        get("/thoughts/{id}") {
            val userId = userResolver.currentUserId(call)
            val id = UUID.fromString(call.parameters["id"])
            val thought = repository.findThought(userId, id) ?: return@get call.respond(HttpStatusCode.NotFound, ErrorResponse("Not found"))
            call.respond(ThoughtResponse.from(thought))
        }

        get("/thoughts/{id}/audio") {
            // Audio bytes go backend -> browser. v1 intentionally avoids presigned URLs.
            val userId = userResolver.currentUserId(call)
            val id = UUID.fromString(call.parameters["id"])
            val thought = repository.findThought(userId, id) ?: return@get call.respond(HttpStatusCode.NotFound, ErrorResponse("Not found"))
            val audio = blobStore.get(thought.audioS3Key)
            call.respondOutputStream(ContentType.parse(audio.contentType), HttpStatusCode.OK) {
                audio.bytes.use { it.copyTo(this) }
            }
        }
    }
}
