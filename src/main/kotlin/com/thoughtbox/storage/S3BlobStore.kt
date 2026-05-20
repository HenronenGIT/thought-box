package com.thoughtbox.storage

import com.thoughtbox.config.S3Config
import com.thoughtbox.domain.StoredAudio
import org.slf4j.LoggerFactory
import software.amazon.awssdk.auth.credentials.AwsBasicCredentials
import software.amazon.awssdk.auth.credentials.StaticCredentialsProvider
import software.amazon.awssdk.core.sync.RequestBody
import software.amazon.awssdk.regions.Region
import software.amazon.awssdk.services.s3.S3Client
import software.amazon.awssdk.services.s3.model.GetObjectRequest
import software.amazon.awssdk.services.s3.model.HeadObjectRequest
import software.amazon.awssdk.services.s3.model.NoSuchKeyException
import software.amazon.awssdk.services.s3.model.PutObjectRequest
import java.io.InputStream
import java.net.URI
import kotlin.system.measureTimeMillis

// AWS SDK implementation. MinIO works by setting endpointOverride through
// S3_ENDPOINT; production leaves endpoint null and talks to AWS S3.
class S3BlobStore(private val config: S3Config) : BlobStore {
    private val logger = LoggerFactory.getLogger(javaClass)
    private val client = S3Client.builder()
        .region(Region.of(config.region))
        .credentialsProvider(StaticCredentialsProvider.create(AwsBasicCredentials.create(config.accessKeyId, config.secretAccessKey)))
        .apply { if (config.endpoint != null) endpointOverride(URI.create(config.endpoint)) }
        .forcePathStyle(config.endpoint != null)
        .build()

    // RequestBody.fromInputStream streams bytes to the SDK without loading full
    // audio into a Kotlin ByteArray.
    override fun put(key: String, contentType: String, contentLength: Long, input: InputStream) {
        var status = 200
        val duration = measureTimeMillis {
            try {
                client.putObject(
                    PutObjectRequest.builder().bucket(config.bucket).key(key).contentType(contentType).build(),
                    RequestBody.fromInputStream(input, contentLength),
                )
            } catch (e: RuntimeException) {
                status = 500
                throw e
            }
        }
        logger.info("external_api provider=s3 operation=put duration_ms={} response_status={}", duration, status)
    }

    override fun get(key: String): StoredAudio {
        val response = client.getObject(GetObjectRequest.builder().bucket(config.bucket).key(key).build())
        return StoredAudio(response, response.response().contentType(), response.response().contentLength())
    }

    override fun exists(key: String): Boolean = try {
        client.headObject(HeadObjectRequest.builder().bucket(config.bucket).key(key).build())
        true
    } catch (_: NoSuchKeyException) {
        false
    }
}
