package com.volna.app.auth.presentation

import com.volna.app.auth.TelegramLoginStart
import com.volna.app.auth.TelegramPollResult
import com.volna.app.auth.VerifyCodeResult
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.AppFailureException
import com.volna.app.domain.model.Client
import com.volna.app.domain.model.ClientId
import com.volna.app.domain.model.Phone
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.test.advanceUntilIdle
import kotlinx.coroutines.test.runCurrent
import kotlinx.coroutines.test.runTest
import kotlinx.datetime.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull

class AuthStoreTelegramTest {
    private val start = TelegramLoginStart(
        pollToken = "poll-secret",
        deepLink = "https://t.me/apex_login_bot?start=abc",
        confirmCode = "K7M3",
        expiresAt = Instant.parse("2026-09-14T12:10:00Z"),
    )

    private fun session(name: String?, isNew: Boolean) = VerifyCodeResult(
        token = "token",
        client = Client(ClientId("11111111-1111-1111-1111-111111111111"), name, Phone("+79991234567"), Instant.parse("2026-09-14T12:00:00Z")),
        isNew = isNew,
    )

    @Test
    fun confirmedLoginAuthenticates() = runTest {
        val repository = AuthStoreDemoTest.FakeAuthRepository().apply {
            telegramStart = Result.success(start)
            telegramPolls.addAll(
                listOf(
                    Result.success(TelegramPollResult.Pending),
                    Result.failure(AppFailureException(AppFailure.NetworkUnavailable)),
                    Result.success(TelegramPollResult.Confirmed(session(name = "Анна", isNew = false))),
                ),
            )
        }
        val store = AuthStore(repository, AuthStoreDemoTest.UnusedProfileRepository, CoroutineScope(coroutineContext))

        store.accept(AuthIntent.TelegramLogin)
        runCurrent()
        val waiting = assertNotNull(store.state.value.telegram)
        assertEquals("K7M3", waiting.confirmCode)
        assertEquals(start.deepLink, waiting.deepLink)

        advanceUntilIdle()

        assertEquals(AuthEffect.Authenticated, store.effects())
        assertNull(store.state.value.telegram)
        // Сетевой сбой посреди опроса не прервал вход.
        assertEquals(3, repository.pollCalls)
    }

    @Test
    fun newClientWithoutNameGoesToNameStep() = runTest {
        val repository = AuthStoreDemoTest.FakeAuthRepository().apply {
            telegramStart = Result.success(start)
            telegramPolls.add(Result.success(TelegramPollResult.Confirmed(session(name = null, isNew = true))))
        }
        val store = AuthStore(repository, AuthStoreDemoTest.UnusedProfileRepository, CoroutineScope(coroutineContext))

        store.accept(AuthIntent.TelegramLogin)
        advanceUntilIdle()

        assertEquals(AuthStep.Name, store.state.value.step)
        assertNull(store.state.value.telegram)
    }

    @Test
    fun expiredLoginReturnsToMethodsWithMessage() = runTest {
        val repository = AuthStoreDemoTest.FakeAuthRepository().apply {
            telegramStart = Result.success(start)
            telegramPolls.add(Result.success(TelegramPollResult.Expired))
        }
        val store = AuthStore(repository, AuthStoreDemoTest.UnusedProfileRepository, CoroutineScope(coroutineContext))

        store.accept(AuthIntent.TelegramLogin)
        advanceUntilIdle()

        assertNull(store.state.value.telegram)
        assertEquals("Время на вход вышло. Попробуйте ещё раз", store.state.value.message)
    }

    @Test
    fun cancelStopsPolling() = runTest {
        val repository = AuthStoreDemoTest.FakeAuthRepository().apply { telegramStart = Result.success(start) }
        val store = AuthStore(repository, AuthStoreDemoTest.UnusedProfileRepository, CoroutineScope(coroutineContext))

        store.accept(AuthIntent.TelegramLogin)
        runCurrent()
        store.accept(AuthIntent.TelegramCancel)
        advanceUntilIdle()

        assertNull(store.state.value.telegram)
        assertEquals(0, repository.pollCalls)
    }

    @Test
    fun rejectsNonTelegramDeepLink() = runTest {
        val repository = AuthStoreDemoTest.FakeAuthRepository().apply {
            telegramStart = Result.success(start.copy(deepLink = "https://evil.example/t.me/bot"))
        }
        val store = AuthStore(repository, AuthStoreDemoTest.UnusedProfileRepository, CoroutineScope(coroutineContext))

        store.accept(AuthIntent.TelegramLogin)
        advanceUntilIdle()

        assertNull(store.state.value.telegram)
        assertEquals("Не удалось начать вход через Telegram. Попробуйте ещё раз", store.state.value.message)
        assertEquals(0, repository.pollCalls)
    }

    @Test
    fun startFailureShowsMessage() = runTest {
        val repository = AuthStoreDemoTest.FakeAuthRepository().apply {
            telegramStart = Result.failure(AppFailureException(AppFailure.NetworkUnavailable))
        }
        val store = AuthStore(repository, AuthStoreDemoTest.UnusedProfileRepository, CoroutineScope(coroutineContext))

        store.accept(AuthIntent.TelegramLogin)
        advanceUntilIdle()

        assertNull(store.state.value.telegram)
        assertEquals("Не удалось начать вход через Telegram. Попробуйте ещё раз", store.state.value.message)
    }
}
