package com.thoughtbox.pipeline

import com.thoughtbox.db.ThoughtRepository
import org.slf4j.LoggerFactory

// Repairs rows left mid-flight by process restart. Node.js mental model: startup
// bootstrap code that requeues jobs stuck in "processing".
class StartupRecovery(private val repository: ThoughtRepository) {
    private val logger = LoggerFactory.getLogger(javaClass)

    fun run() {
        val recovered = repository.recoverStuckRows()
        logger.info("startup_recovery recovered_rows={}", recovered)
    }
}
