package com.volna.app.auth.presentation

import com.volna.app.auth.AuthMethods
import com.volna.app.auth.AuthRepository
import com.volna.app.auth.RequestCodeResult
import com.volna.app.auth.VerifyCodeResult
import com.volna.app.core.error.ApiErrorCode
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.AppFailureException
import com.volna.app.core.ui.ActionStatus
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.Client
import com.volna.app.domain.model.ClientId
import com.volna.app.domain.model.Phone
import com.volna.app.profile.ProfileRepository
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.yield
import kotlinx.datetime.Instant
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs
import kotlin.test.assertNull

class AuthStoreDemoTest {
    @Test
    fun loadsAuthMethodsOnce() = runTest {
        val repository = FakeAuthRepository(methods = Result.success(AuthMethods(sms = false, demo = true, telegramBotUsername = null)))
        val store = AuthStore(repository, UnusedProfileRepository, CoroutineScope(coroutineContext))

        store.accept(AuthIntent.LoadMethods)
        yield()
        store.accept(AuthIntent.LoadMethods)
        yield()

        val methods = assertIs<Loadable.Content<AuthMethods>>(store.state.value.methods)
        assertEquals(AuthMethods(sms = false, demo = true, telegramBotUsername = null), methods.value)
        assertEquals(1, repository.methodsCalls)
    }

    @Test
    fun methodsFailureCanBeRetried() = runTest {
        val repository = FakeAuthRepository(methods = Result.failure(AppFailureException(AppFailure.NetworkUnavailable)))
        val store = AuthStore(repository, UnusedProfileRepository, CoroutineScope(coroutineContext))

        store.accept(AuthIntent.LoadMethods)
        yield()
        assertIs<Loadable.Error>(store.state.value.methods)

        repository.methods = Result.success(AuthMethods(sms = true, demo = true, telegramBotUsername = null))
        store.accept(AuthIntent.LoadMethods)
        yield()
        assertIs<Loadable.Content<AuthMethods>>(store.state.value.methods)
        assertEquals(2, repository.methodsCalls)
    }

    @Test
    fun demoLoginAuthenticatesWithoutNameStep() = runTest {
        val repository = FakeAuthRepository()
        val store = AuthStore(repository, UnusedProfileRepository, CoroutineScope(coroutineContext))

        store.accept(AuthIntent.DemoLogin)
        val effect = store.effects()

        assertEquals(AuthEffect.Authenticated, effect)
        assertEquals(AuthStep.Phone, store.state.value.step)
        assertEquals(ActionStatus.Idle, store.state.value.actionStatus)
        assertEquals(1, repository.demoCalls)
    }

    @Test
    fun demoLoginLimitShowsMessage() = runTest {
        val repository = FakeAuthRepository(
            demo = Result.failure(AppFailureException(AppFailure.Api(ApiErrorCode.TooManyRequests, "limit"))),
        )
        val store = AuthStore(repository, UnusedProfileRepository, CoroutineScope(coroutineContext))

        store.accept(AuthIntent.DemoLogin)
        yield()

        assertEquals("Демо-вход сейчас недоступен. Попробуйте позже", store.state.value.message)
        assertEquals(ActionStatus.Idle, store.state.value.actionStatus)
    }

    @Test
    fun resetKeepsLoadedMethods() = runTest {
        val repository = FakeAuthRepository(methods = Result.success(AuthMethods(sms = false, demo = true, telegramBotUsername = "apex_bot")))
        val store = AuthStore(repository, UnusedProfileRepository, CoroutineScope(coroutineContext))

        store.accept(AuthIntent.LoadMethods)
        yield()
        store.accept(AuthIntent.Reset)

        assertIs<Loadable.Content<AuthMethods>>(store.state.value.methods)
        assertNull(store.state.value.message)
    }

    private class FakeAuthRepository(
        var methods: Result<AuthMethods> = Result.success(AuthMethods(sms = true, demo = true, telegramBotUsername = null)),
        private val demo: Result<VerifyCodeResult> = Result.success(
            VerifyCodeResult(
                token = "token",
                client = Client(
                    id = ClientId("11111111-1111-1111-1111-111111111111"),
                    name = "Гость",
                    phone = Phone("+70001234567"),
                    createdAt = Instant.parse("2026-09-14T10:00:00Z"),
                    isDemo = true,
                    demoExpiresAt = Instant.parse("2026-09-15T10:00:00Z"),
                ),
                isNew = false,
            ),
        ),
    ) : AuthRepository {
        var methodsCalls = 0
            private set
        var demoCalls = 0
            private set

        override suspend fun authMethods(): Result<AuthMethods> {
            methodsCalls++
            return methods
        }

        override suspend fun demoLogin(): Result<VerifyCodeResult> {
            demoCalls++
            return demo
        }

        override suspend fun requestCode(phone: Phone): Result<RequestCodeResult> = error("not used")
        override suspend fun verifyCode(phone: Phone, code: String): Result<VerifyCodeResult> = error("not used")
        override suspend fun logout(): Result<Unit> = error("not used")
    }

    private object UnusedProfileRepository : ProfileRepository {
        override suspend fun getProfile(): Result<Client> = error("not used")
        override suspend fun updateName(name: String): Result<Client> = error("not used")
        override suspend fun deleteAccount(): Result<Unit> = error("not used")
        override suspend fun requestPhoneChangeCode(newPhone: Phone): Result<RequestCodeResult> = error("not used")
        override suspend fun confirmPhoneChange(newPhone: Phone, code: String): Result<Client> = error("not used")
    }
}
