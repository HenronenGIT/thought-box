plugins {
    alias(libs.plugins.kotlin.jvm)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.shadow)
    application
}

group = "com.thoughtbox"
version = "0.1.0"

kotlin {
    jvmToolchain(21)
}

application {
    mainClass.set("com.thoughtbox.ApplicationKt")
}

dependencies {
    implementation(libs.ktor.server.core)
    implementation(libs.ktor.server.netty)
    implementation(libs.ktor.server.content.negotiation)
    implementation(libs.ktor.server.cors)
    implementation(libs.ktor.server.status.pages)
    implementation(libs.ktor.serialization.json)
    implementation(libs.ktor.client.core)
    implementation(libs.ktor.client.cio)
    implementation(libs.ktor.client.content.negotiation)
    implementation(libs.logback)
    implementation(libs.logstash)
    implementation(libs.flyway)
    implementation(libs.hikari)
    implementation(libs.kotliquery)
    implementation(libs.postgres)
    implementation(libs.aws.s3)
    implementation(libs.sentry)

    testImplementation(libs.junit)
    testImplementation(libs.kotest.assertions)
    testImplementation(libs.mockk)
    testRuntimeOnly(libs.junit.platform.launcher)
}

tasks.test {
    useJUnitPlatform()
}

fun loadDotEnv(file: File): Map<String, String> {
    if (!file.exists()) return emptyMap()
    return file.readLines()
        .map { it.trim() }
        .filter { it.isNotBlank() && !it.startsWith("#") && it.contains("=") }
        .associate { line ->
            val key = line.substringBefore("=").trim()
            val value = line.substringAfter("=").trim().trim('"', '\'')
            key to value
        }
}

tasks.named<JavaExec>("run") {
    environment(loadDotEnv(rootProject.file(".env")))
}

tasks.register<JavaExec>("runDev") {
    group = "application"
    description = "Runs the backend locally with environment variables from .env."
    classpath = sourceSets["main"].runtimeClasspath
    mainClass.set(application.mainClass)
    environment(loadDotEnv(rootProject.file(".env")))
}

tasks.shadowJar {
    archiveFileName.set("thought-box.jar")
    mergeServiceFiles()
}
