package com.thoughtbox

import com.thoughtbox.config.Config
import com.thoughtbox.db.ThoughtRepository
import com.thoughtbox.db.dataSource
import com.thoughtbox.db.runMigrations
import com.thoughtbox.enrichment.OpenAiEnricher
import com.thoughtbox.http.configureRoutes
import com.thoughtbox.observability.configureObservability
import com.thoughtbox.pipeline.StartupRecovery
import com.thoughtbox.pipeline.ThoughtPipeline
import com.thoughtbox.storage.S3BlobStore
import com.thoughtbox.transcription.OpenAiTranscriber
import com.thoughtbox.user.SeededUserResolver
import io.ktor.serialization.kotlinx.json.json
import io.ktor.server.application.Application
import io.ktor.server.application.ApplicationStopped
import io.ktor.server.application.install
import io.ktor.server.engine.embeddedServer
import io.ktor.server.netty.Netty
import io.ktor.server.plugins.contentnegotiation.ContentNegotiation
import io.ktor.server.plugins.cors.routing.CORS
import io.ktor.http.HttpHeaders
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.serialization.json.Json
import org.slf4j.LoggerFactory
import java.net.URI

// Application entrypoint. Node.js mental model: this is closest to the file
// where you create an Express app, install middleware, wire services, then call
// app.listen(PORT). In Ktor, embeddedServer(...).start(...) owns that lifecycle.
fun main() {
    val config = Config.fromEnv()
    embeddedServer(Netty, port = config.port, host = "0.0.0.0") {
        module(config)
    }.start(wait = true)
}

// Ktor application module. This is the composition root: concrete classes are
// created here, while routes/pipeline code mostly depend on interfaces.
fun Application.module(config: Config = Config.fromEnv()) {
    val logger = LoggerFactory.getLogger("startup")
    logger.info(
        "startup app_env={} postgres_host={} s3_bucket={} s3_endpoint={}",
        config.appEnv,
        config.database.host,
        config.s3.bucket,
        config.s3.endpoint ?: "aws",
    )

    // Installs cross-cutting behavior first: logging, exception handling, Sentry.
    configureObservability(config)
    install(ContentNegotiation) { json(Json { ignoreUnknownKeys = true }) }
    install(CORS) {
        allowHeader(HttpHeaders.ContentType)
        allowHeader(HttpHeaders.Authorization)
        allowHeader("X-Correlation-Id")
        allowMethod(io.ktor.http.HttpMethod.Get)
        allowMethod(io.ktor.http.HttpMethod.Post)
        config.corsAllowedOrigins.forEach { origin ->
            val uri = URI.create(origin)
            allowHost(uri.host, schemes = listOf(uri.scheme))
        }
    }

    // Flyway applies schema changes on boot so local/prod DBs stay aligned.
    runMigrations(config.database)
    val ds = dataSource(config.database)
    val repository = ThoughtRepository(ds)
    StartupRecovery(repository).run()

    // Interface-backed modules. Node.js mental model: these are service objects
    // you might put in req.app.locals or pass into route factory functions.
    val blobStore = S3BlobStore(config.s3)
    val transcriber = OpenAiTranscriber(config.openAiApiKey, blobStore)
    val enricher = OpenAiEnricher(config.openAiApiKey)
    val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    val pipeline = ThoughtPipeline(repository, transcriber, enricher, scope)
    // Starts background coroutine worker. Node.js mental model: a long-running
    // async loop started beside the HTTP server, but tied to app shutdown.
    pipeline.start()

    configureRoutes(config, repository, blobStore, SeededUserResolver())

    environment.monitor.subscribe(ApplicationStopped) {
        pipeline.stop()
        ds.close()
    }
}
