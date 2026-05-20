package com.thoughtbox.domain

import io.kotest.assertions.throwables.shouldThrow
import io.kotest.matchers.shouldBe
import org.junit.jupiter.api.Test

class StatusTest {
    @Test
    fun `legal transitions`() {
        Status.Pending.next(StatusEvent.StartTranscription) shouldBe Status.Transcribing
        Status.Transcribing.next(StatusEvent.TranscriptionSucceeded) shouldBe Status.Enriching
        Status.Transcribing.next(StatusEvent.TranscriptionFailedTerminal) shouldBe Status.FailedTranscription
        Status.Enriching.next(StatusEvent.EnrichmentSucceeded) shouldBe Status.Done
        Status.Enriching.next(StatusEvent.EnrichmentFailedTerminal) shouldBe Status.FailedEnrichment
        Status.Transcribing.next(StatusEvent.Recover) shouldBe Status.Pending
    }

    @Test
    fun `illegal transitions throw`() {
        shouldThrow<IllegalStateException> {
            Status.Done.next(StatusEvent.StartTranscription)
        }
    }
}

