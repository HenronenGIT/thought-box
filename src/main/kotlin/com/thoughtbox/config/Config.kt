package com.thoughtbox.config

import java.net.URI

// Typed runtime configuration. Node.js mental model: instead of reading
// process.env everywhere, parse and validate it once, then pass this object in.
data class Config(
    val appEnv: String,
    val port: Int,
    val database: DatabaseConfig,
    val corsAllowedOrigins: List<String>,
    val s3: S3Config,
    val openAiApiKey: String,
    val sentryDsn: String?,
    val limits: LimitsConfig,
) {
    companion object {
        // Accepting a Map makes this easy to unit test without mutating real env.
        fun fromEnv(env: Map<String, String> = System.getenv()): Config {
            fun required(name: String) = env[name]?.trim()?.takeIf { it.isNotBlank() }
                ?: throw IllegalArgumentException("Missing required env var: $name")
            fun optionalInt(name: String, default: Int) = env[name]?.trim()?.toIntOrNull() ?: default
            fun optionalLong(name: String, default: Long) = env[name]?.trim()?.toLongOrNull() ?: default

            return Config(
                appEnv = env["APP_ENV"]?.trim()?.takeIf { it.isNotBlank() } ?: "dev",
                port = optionalInt("PORT", 8080),
                database = DatabaseConfig(
                    url = required("DATABASE_URL"),
                    user = required("DATABASE_USER"),
                    password = required("DATABASE_PASSWORD"),
                ),
                corsAllowedOrigins = (env["CORS_ALLOWED_ORIGINS"] ?: "http://localhost:3000")
                    .split(",")
                    .map { it.trim() }
                    .filter { it.isNotBlank() },
                s3 = S3Config(
                    bucket = required("S3_BUCKET"),
                    region = required("S3_REGION"),
                    endpoint = env["S3_ENDPOINT"]?.trim()?.takeIf { it.isNotBlank() },
                    accessKeyId = required("AWS_ACCESS_KEY_ID"),
                    secretAccessKey = required("AWS_SECRET_ACCESS_KEY"),
                ),
                openAiApiKey = required("OPENAI_API_KEY"),
                sentryDsn = env["SENTRY_DSN"]?.trim()?.takeIf { it.isNotBlank() },
                limits = LimitsConfig(
                    maxDurationMs = optionalLong("MAX_THOUGHT_DURATION_MS", 60_000),
                    minDurationMs = optionalLong("MIN_THOUGHT_DURATION_MS", 1_000),
                    maxSizeBytes = optionalLong("MAX_THOUGHT_SIZE_BYTES", 10 * 1024 * 1024),
                ),
            )
        }
    }
}

// JDBC URLs include credentials in some deployments; only host is exposed for logs.
data class DatabaseConfig(val url: String, val user: String, val password: String) {
    val host: String = runCatching { URI(url.removePrefix("jdbc:")).host }.getOrNull() ?: "unknown"
}

data class S3Config(
    val bucket: String,
    val region: String,
    val endpoint: String?,
    val accessKeyId: String,
    val secretAccessKey: String,
)

data class LimitsConfig(
    val maxDurationMs: Long,
    val minDurationMs: Long,
    val maxSizeBytes: Long,
)
