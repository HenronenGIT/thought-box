package com.thoughtbox.user

import com.thoughtbox.domain.SeededUserId
import io.ktor.server.application.ApplicationCall
import java.util.UUID

// Future auth swap point. Node.js mental model: req.user.id resolver/middleware,
// except v1 always returns one known user.
interface UserResolver {
    fun currentUserId(call: ApplicationCall): UUID
}

// v1 has no login, so every request belongs to the seeded user row.
class SeededUserResolver : UserResolver {
    override fun currentUserId(call: ApplicationCall): UUID = SeededUserId
}
