package com.thoughtbox.pipeline

import com.thoughtbox.db.ThoughtRepository
import com.thoughtbox.domain.AudioBlob
import com.thoughtbox.domain.Status
import com.thoughtbox.domain.Thought
import com.thoughtbox.enrichment.Enricher
import com.thoughtbox.transcription.Transcriber
import io.sentry.Sentry
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import org.slf4j.LoggerFactory

// In-process background worker. Node.js mental model: an async while loop that
// polls the DB, calls external APIs, and updates status. Kotlin coroutines make
// the sleeps/non-blocking calls cheap without a separate queue system.
class ThoughtPipeline(
    private val repository: ThoughtRepository,
    private val transcriber: Transcriber,
    private val enricher: Enricher,
    private val scope: CoroutineScope,
) {
    private val logger = LoggerFactory.getLogger(javaClass)
    private var job: Job? = null

    // launch creates a coroutine owned by the application scope.
    fun start() {
        job = scope.launch {
            while (isActive) {
                val didWork = processOneTranscription() || processOneEnrichment()
                if (!didWork) delay(1_000)
            }
        }
    }

    fun stop() {
        job?.cancel()
    }

    // One pending thought: claim row, call Whisper, move to enrichment or retry.
    private suspend fun processOneTranscription(): Boolean {
        val thought = repository.nextPending() ?: return false
        logger.info("pipeline_transition thought_id={} from_status=pending to_status=transcribing attempts={}", thought.id, thought.attempts)
        try {
            val result = transcriber.transcribe(AudioBlob(thought.audioS3Key, thought.mimeType, thought.sizeBytes))
            repository.markTranscribed(thought.id, result.text)
            logger.info("pipeline_transition thought_id={} from_status=transcribing to_status=enriching attempts=0", thought.id)
        } catch (e: Exception) {
            handleFailure(thought, Status.FailedTranscription, e)
        }
        return true
    }

    // One enrichment-ready thought: call GPT, insert enrichment, finish or retry.
    private suspend fun processOneEnrichment(): Boolean {
        val thought = repository.nextEnriching() ?: return false
        try {
            val transcript = thought.transcript ?: error("Missing transcript")
            val enrichment = enricher.enrich(thought.id, transcript)
            repository.markEnriched(thought.id, enrichment)
            logger.info("pipeline_transition thought_id={} from_status=enriching to_status=done attempts=0", thought.id)
        } catch (e: Exception) {
            handleFailure(thought, Status.FailedEnrichment, e)
        }
        return true
    }

    // Shared retry/fail logic for transcription and enrichment.
    private suspend fun handleFailure(thought: Thought, terminalStatus: Status, error: Exception) {
        val attempts = thought.attempts + 1
        when (val decision = retryDecision(thought.attempts)) {
            RetryDecision.Fail -> {
                repository.recordFailure(thought.id, terminalStatus, attempts, error.message ?: error.javaClass.simpleName)
                logger.warn(
                    "pipeline_transition thought_id={} from_status={} to_status={} attempts={} error_class={}",
                    thought.id,
                    thought.status.wireValue(),
                    terminalStatus.wireValue(),
                    attempts,
                    error.javaClass.simpleName,
                )
                Sentry.captureException(error)
            }
            is RetryDecision.RetryAfter -> {
                val retryStatus = if (terminalStatus == Status.FailedTranscription) Status.Pending else Status.Enriching
                repository.recordRetry(thought.id, retryStatus, attempts, error.message ?: error.javaClass.simpleName)
                logger.warn("pipeline_retry thought_id={} attempts={} error_class={}", thought.id, attempts, error.javaClass.simpleName)
                delay(decision.delay)
            }
        }
    }
}
