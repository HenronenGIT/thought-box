package com.thoughtbox.transcription

import com.thoughtbox.domain.AudioBlob

data class TranscriptionResult(val text: String, val model: String)

// Provider boundary for speech-to-text. Node.js mental model: an interface you
// would fake in tests and implement with an OpenAI HTTP client module.
interface Transcriber {
    suspend fun transcribe(audio: AudioBlob): TranscriptionResult
}
