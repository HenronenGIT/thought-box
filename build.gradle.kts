plugins {
    alias(libs.plugins.kotlin.jvm)
    alias(libs.plugins.kotlin.serialization)
    alias(libs.plugins.shadow)
    alias(libs.plugins.sentry.jvm)
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

sentry {
    includeSourceContext.set(true)

    org.set("hmaronen")
    projectName.set("kotlin")
    authToken.set(System.getenv("SENTRY_AUTH_TOKEN"))
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
}

tasks.test {
    useJUnitPlatform()
}

tasks.shadowJar {
    archiveFileName.set("thought-box.jar")
    mergeServiceFiles()
}
