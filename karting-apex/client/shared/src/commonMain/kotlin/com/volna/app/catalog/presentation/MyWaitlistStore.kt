package com.volna.app.catalog.presentation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.volna.app.catalog.MyWaitlistEntry
import com.volna.app.catalog.WaitlistRepository
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.asAppFailure
import com.volna.app.core.logging.AppLogger
import com.volna.app.core.mvi.MviStore
import com.volna.app.domain.model.SlotId
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

/**
 * «Мои очереди» в профиле. Раздел вспомогательный: не загрузился — не показываем, без
 * отдельного экрана ошибки (сессию и так проверяет профиль).
 */
data class MyWaitlistState(
    val entries: List<MyWaitlistEntry> = emptyList(),
    /** Заезды, из очереди на которые человек выходит прямо сейчас. */
    val leaving: Set<SlotId> = emptySet(),
    val message: String? = null,
)

sealed interface MyWaitlistIntent {
    data object Load : MyWaitlistIntent
    data class Leave(val slotId: SlotId) : MyWaitlistIntent
}

/** Эффектов нет: истёкшую сессию обрабатывает стор профиля на том же экране. */
sealed interface MyWaitlistEffect

class MyWaitlistStore(
    private val waitlistRepository: WaitlistRepository,
    scope: CoroutineScope? = null,
) : ViewModel(), MviStore<MyWaitlistState, MyWaitlistIntent, MyWaitlistEffect> {
    private val mutableState = MutableStateFlow(MyWaitlistState())
    private val storeScope = scope ?: viewModelScope

    override val state: StateFlow<MyWaitlistState> = mutableState

    override fun accept(intent: MyWaitlistIntent) {
        when (intent) {
            MyWaitlistIntent.Load -> load()
            is MyWaitlistIntent.Leave -> leave(intent.slotId)
        }
    }

    override suspend fun effects(): MyWaitlistEffect = awaitCancellation()

    private fun load() {
        storeScope.launch {
            waitlistRepository.mine().fold(
                onSuccess = { entries -> mutableState.update { it.copy(entries = entries) } },
                onFailure = { failure -> AppLogger.e(failure, "Failed to load my waitlist") },
            )
        }
    }

    private fun leave(slotId: SlotId) {
        if (slotId in mutableState.value.leaving) return
        mutableState.update { it.copy(leaving = it.leaving + slotId, message = null) }
        storeScope.launch {
            val result = waitlistRepository.leave(slotId)
            mutableState.update { current ->
                result.fold(
                    onSuccess = {
                        current.copy(entries = current.entries.filterNot { it.slotId == slotId }, leaving = current.leaving - slotId)
                    },
                    onFailure = { failure ->
                        AppLogger.e(failure, "Failed to leave waitlist from profile")
                        current.copy(leaving = current.leaving - slotId, message = failure.leaveMessage())
                    },
                )
            }
        }
    }
}

private fun Throwable.leaveMessage(): String = when (val failure = asAppFailure()) {
    is AppFailure.Api -> failure.message
    AppFailure.NetworkUnavailable, AppFailure.Timeout -> "Нет соединения. Проверьте интернет и попробуйте снова."
    else -> "Не получилось выйти из очереди. Попробуйте ещё раз."
}

private suspend fun awaitCancellation(): Nothing = kotlinx.coroutines.awaitCancellation()
