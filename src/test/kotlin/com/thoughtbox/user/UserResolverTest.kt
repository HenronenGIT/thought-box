package com.thoughtbox.user

import com.thoughtbox.domain.SeededUserId
import io.kotest.matchers.shouldBe
import io.mockk.mockk
import org.junit.jupiter.api.Test

class UserResolverTest {
    @Test
    fun `returns seeded user id`() {
        SeededUserResolver().currentUserId(mockk(relaxed = true)) shouldBe SeededUserId
    }
}

