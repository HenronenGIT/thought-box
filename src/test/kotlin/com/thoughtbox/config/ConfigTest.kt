package com.thoughtbox.config

import io.kotest.assertions.throwables.shouldThrow
import io.kotest.matchers.shouldBe
import org.junit.jupiter.api.Test

class ConfigTest {
    private val required = mapOf(
        "DATABASE_URL" to "jdbc:postgresql://localhost:5432/thoughts_dev",
        "DATABASE_USER" to "thoughts",
        "DATABASE_PASSWORD" to "thoughts",
        "S3_BUCKET" to "thoughts-dev",
        "S3_REGION" to "us-east-1",
        "AWS_ACCESS_KEY_ID" to "dev",
        "AWS_SECRET_ACCESS_KEY" to "dev",
        "OPENAI_API_KEY" to "dev",
    )

    @Test
    fun `applies defaults`() {
        val config = Config.fromEnv(required)

        config.appEnv shouldBe "dev"
        config.port shouldBe 8080
        config.limits.maxDurationMs shouldBe 60_000
        config.limits.minDurationMs shouldBe 1_000
        config.limits.maxSizeBytes shouldBe 10 * 1024 * 1024
    }

    @Test
    fun `rejects missing required values`() {
        shouldThrow<IllegalArgumentException> {
            Config.fromEnv(required - "DATABASE_URL")
        }.message shouldBe "Missing required env var: DATABASE_URL"
    }

    @Test
    fun `trims environment values`() {
        val config = Config.fromEnv(
            required + mapOf(
                "APP_ENV" to " prod ",
                "PORT" to " 9090\n",
                "DATABASE_URL" to " jdbc:postgresql://localhost:5432/thoughts_dev?sslmode=require\n",
                "S3_ENDPOINT" to " http://localhost:4566 ",
                "SENTRY_DSN" to " https://example.com/1\n",
            ),
        )

        config.appEnv shouldBe "prod"
        config.port shouldBe 9090
        config.database.url shouldBe "jdbc:postgresql://localhost:5432/thoughts_dev?sslmode=require"
        config.s3.endpoint shouldBe "http://localhost:4566"
        config.sentryDsn shouldBe "https://example.com/1"
    }
}
