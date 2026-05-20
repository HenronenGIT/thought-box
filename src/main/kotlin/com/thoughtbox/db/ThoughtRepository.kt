package com.thoughtbox.db

import com.thoughtbox.domain.Category
import com.thoughtbox.domain.Status
import com.thoughtbox.domain.Thought
import com.thoughtbox.domain.ThoughtEnrichment
import kotliquery.Row
import kotliquery.Session
import kotliquery.queryOf
import kotliquery.sessionOf
import java.sql.Timestamp
import java.time.Instant
import java.util.UUID
import javax.sql.DataSource

// Repository owns raw SQL. Node.js mental model: a module wrapping node-postgres
// queries; route handlers call methods here instead of embedding SQL inline.
class ThoughtRepository(private val dataSource: DataSource) {
    // Insert happens only after audio already exists in BlobStore.
    fun insertThought(id: UUID, userId: UUID, audioKey: String, mimeType: String, durationMs: Long, sizeBytes: Long): Thought =
        sessionOf(dataSource).use { session ->
            session.run(
                queryOf(
                    """
                    insert into thoughts (id, user_id, audio_s3_key, mime_type, duration_ms, size_bytes, status)
                    values (?, ?, ?, ?, ?, ?, 'pending')
                    returning *, null::text as category, null::text as tags_csv,
                        null::text as title, null::text as summary, null::text as model, null::text as prompt_version
                    """.trimIndent(),
                    id, userId, audioKey, mimeType, durationMs, sizeBytes,
                ).map(::mapThought).asSingle
            ) ?: error("insert failed")
        }

    // User id is part of the lookup so future auth can enforce row ownership here.
    fun findThought(userId: UUID, id: UUID): Thought? = sessionOf(dataSource).use { session ->
        session.run(
            queryOf(
                """
                select t.*, e.category, array_to_string(e.tags, ',') as tags_csv, e.title, e.summary, e.model, e.prompt_version
                from thoughts t
                left join thought_enrichments e on e.thought_id = t.id
                where t.user_id = ? and t.id = ?
                """.trimIndent(),
                userId, id,
            ).map(::mapThought).asSingle
        )
    }

    // Cursor is created_at. Optional category/tag filters join through enrichment.
    fun listThoughts(userId: UUID, limit: Int, before: Instant?, category: Category?, tag: String?): List<Thought> =
        sessionOf(dataSource).use { session ->
            val params = mutableListOf<Any>(userId)
            val filters = mutableListOf("t.user_id = ?")
            if (before != null) {
                filters += "t.created_at < ?"
                params += Timestamp.from(before)
            }
            if (category != null) {
                filters += "e.category = ?"
                params += category.name
            }
            if (!tag.isNullOrBlank()) {
                filters += "? = any(e.tags)"
                params += tag
            }
            params += limit
            session.run(
                queryOf(
                    """
                    select t.*, e.category, array_to_string(e.tags, ',') as tags_csv, e.title, e.summary, e.model, e.prompt_version
                    from thoughts t
                    left join thought_enrichments e on e.thought_id = t.id
                    where ${filters.joinToString(" and ")}
                    order by t.created_at desc
                    limit ?
                    """.trimIndent(),
                    *params.toTypedArray(),
                ).map(::mapThought).asList
            )
        }

    // FOR UPDATE SKIP LOCKED prevents two workers from claiming the same row.
    // Even with one worker in v1, this is the safe Postgres pattern.
    fun nextPending(): Thought? = sessionOf(dataSource).use { session ->
        session.transaction { tx ->
            val thought = tx.run(
                queryOf(
                    """
                    select *, null::text as category, null::text as tags_csv,
                        null::text as title, null::text as summary, null::text as model, null::text as prompt_version
                    from thoughts
                    where status = 'pending'
                    order by created_at asc
                    for update skip locked
                    limit 1
                    """.trimIndent(),
                ).map(::mapThought).asSingle
            )
            if (thought != null) updateStatus(tx, thought.id, Status.Transcribing)
            thought
        }
    }

    // Finds rows that have transcript text but no enrichment row yet.
    fun nextEnriching(): Thought? = sessionOf(dataSource).use { session ->
        session.transaction { tx ->
            tx.run(
                queryOf(
                    """
                    select *, null::text as category, null::text as tags_csv,
                        null::text as title, null::text as summary, null::text as model, null::text as prompt_version
                    from thoughts
                    where status = 'enriching'
                        and transcript is not null
                        and not exists (
                            select 1 from thought_enrichments where thought_enrichments.thought_id = thoughts.id
                        )
                    order by created_at asc
                    for update skip locked
                    limit 1
                    """.trimIndent(),
                ).map(::mapThought).asSingle
            )
        }
    }

    // Successful transcription moves the row to enrichment-ready state.
    fun markTranscribed(id: UUID, transcript: String) = sessionOf(dataSource).use { session ->
        session.run(
            queryOf(
                """
                update thoughts
                set transcript = ?, status = 'enriching', attempts = 0, last_error = null,
                    transcribed_at = now(), updated_at = now()
                where id = ?
                """.trimIndent(),
                transcript, id,
            ).asUpdate
        )
    }

    // Enrichment insert and status update are one DB transaction.
    fun markEnriched(id: UUID, enrichment: ThoughtEnrichment) = sessionOf(dataSource).use { session ->
        session.transaction { tx ->
            tx.run(
                queryOf(
                    """
                    insert into thought_enrichments
                    (thought_id, category, tags, title, summary, model, prompt_version)
                    values (?, ?, string_to_array(?, ','), ?, ?, ?, ?)
                    """.trimIndent(),
                    id,
                    enrichment.category.name,
                    enrichment.tags.joinToString(","),
                    enrichment.title,
                    enrichment.summary,
                    enrichment.model,
                    enrichment.promptVersion,
                ).asUpdate
            )
            updateStatus(tx, id, Status.Done)
        }
    }

    // Terminal failures keep diagnostic state on the thought row.
    fun recordFailure(id: UUID, status: Status, attempts: Int, error: String) = sessionOf(dataSource).use { session ->
        session.run(
            queryOf(
                """
                update thoughts
                set status = ?, attempts = ?, last_error = ?, last_attempt_at = now(), updated_at = now()
                where id = ?
                """.trimIndent(),
                status.wireValue(), attempts, error.take(1000), id,
            ).asUpdate
        )
    }

    // Retry failures return the row to a recoverable status.
    fun recordRetry(id: UUID, retryStatus: Status, attempts: Int, error: String) = sessionOf(dataSource).use { session ->
        session.run(
            queryOf(
                """
                update thoughts
                set status = ?, attempts = ?, last_error = ?, last_attempt_at = now(), updated_at = now()
                where id = ?
                """.trimIndent(),
                retryStatus.wireValue(), attempts, error.take(1000), id,
            ).asUpdate
        )
    }

    // Boot-time repair for work interrupted by deploy/restart/crash.
    fun recoverStuckRows() = sessionOf(dataSource).use { session ->
        session.run(
            queryOf(
                """
                update thoughts
                set status = case when status = 'transcribing' then 'pending' else status end,
                    updated_at = now()
                where status in ('transcribing', 'enriching')
                """.trimIndent(),
            ).asUpdate
        )
    }

    private fun updateStatus(session: Session, id: UUID, status: Status) {
        session.run(
            queryOf("update thoughts set status = ?, updated_at = now() where id = ?", status.wireValue(), id).asUpdate
        )
    }

    private fun mapThought(row: Row): Thought {
        val tags = row.stringOrNull("tags_csv")?.split(",")?.filter { it.isNotBlank() }.orEmpty()
        val enrichment = row.stringOrNull("category")?.let {
            ThoughtEnrichment(
                category = Category.valueOf(it),
                tags = tags,
                title = row.string("title"),
                summary = row.string("summary"),
                model = row.string("model"),
                promptVersion = row.string("prompt_version"),
            )
        }
        return Thought(
            id = row.uuid("id"),
            userId = row.uuid("user_id"),
            createdAt = row.zonedDateTime("created_at").toInstant(),
            updatedAt = row.zonedDateTime("updated_at").toInstant(),
            audioS3Key = row.string("audio_s3_key"),
            mimeType = row.string("mime_type"),
            durationMs = row.longOrNull("duration_ms"),
            sizeBytes = row.long("size_bytes"),
            transcript = row.stringOrNull("transcript"),
            status = Status.fromDb(row.string("status")),
            attempts = row.int("attempts"),
            lastError = row.stringOrNull("last_error"),
            transcribedAt = row.zonedDateTimeOrNull("transcribed_at")?.toInstant(),
            enrichment = enrichment,
        )
    }
}
