package com.thoughtbox.storage

import com.thoughtbox.domain.StoredAudio
import java.io.InputStream

// Object-storage boundary. Node.js mental model: a small interface around S3
// getObject/putObject so routes and transcribers do not know provider details.
interface BlobStore {
    fun put(key: String, contentType: String, contentLength: Long, input: InputStream)
    fun get(key: String): StoredAudio
    fun exists(key: String): Boolean
}
