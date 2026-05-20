package com.thoughtbox.observability

import com.thoughtbox.config.Config
import com.thoughtbox.http.ErrorResponse
import io.ktor.http.HttpStatusCode
import io.ktor.server.application.Application
import io.ktor.server.application.ApplicationCall
import io.ktor.server.application.ApplicationCallPipeline
import io.ktor.server.application.call
import io.ktor.server.application.install
import io.ktor.server.plugins.statuspages.StatusPages
import io.ktor.server.request.httpMethod
import io.ktor.server.request.path
import io.ktor.server.response.respond
import io.sentry.Sentry
import org.slf4j.LoggerFactory
import org.slf4j.MDC
import java.util.UUID

// Cross-cutting runtime instrumentation. Node.js mental model: request logging
// middleware plus an error-handling middleware wired to Sentry.
fun Application.configureObservability(config: Config) {
    if (config.sentryDsn != null) {
        Sentry.init { options ->
            options.dsn = config.sentryDsn
            options.environment = config.appEnv
        }
    }

    intercept(ApplicationCallPipeline.Monitoring) {
        // MDC is thread-local logging context. Logback includes correlation_id in
        // JSON logs emitted during this request.
        val correlationId = call.request.headers["X-Correlation-Id"] ?: UUID.randomUUID().toString()
        MDC.put("correlation_id", correlationId)
        val logger = LoggerFactory.getLogger("http")
        val started = System.nanoTime()
        var status = 500
        try {
            proceed()
            status = call.response.status()?.value ?: 200
        } finally {
            val duration = (System.nanoTime() - started) / 1_000_000
            logger.info(
                "http_request method={} path={} status={} duration_ms={} correlation_id={}",
                call.request.httpMethod.value,
                call.request.path(),
                status,
                duration,
                correlationId,
            )
            MDC.clear()
        }
    }

    install(StatusPages) {
        // StatusPages is Ktor's structured exception handler.
        exception<IllegalArgumentException> { call: ApplicationCall, cause: IllegalArgumentException ->
            call.respond(HttpStatusCode.BadRequest, ErrorResponse(cause.message ?: "Bad request"))
        }
        exception<Throwable> { call: ApplicationCall, cause: Throwable ->
            Sentry.configureScope { scope -> scope.setTag("correlation_id", MDC.get("correlation_id") ?: "missing") }
            Sentry.captureException(cause)
            call.respond(HttpStatusCode.InternalServerError, ErrorResponse("Internal server error"))
        }
    }
}
