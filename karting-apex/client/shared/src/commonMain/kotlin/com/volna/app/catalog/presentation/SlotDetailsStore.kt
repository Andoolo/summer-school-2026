package com.volna.app.catalog.presentation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.volna.app.catalog.DisabledWaitlistRepository
import com.volna.app.catalog.SlotRepository
import com.volna.app.catalog.WaitlistRepository
import com.volna.app.catalog.WaitlistStatus
import com.volna.app.core.error.ApiErrorCode
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.asAppFailure
import com.volna.app.core.logging.AppLogger
import com.volna.app.core.mvi.MviStore
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.LeaderboardEntry
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class SlotDetailsState(
    val slot: Loadable<Slot> = Loadable.Initial,
    val showRouteMap: Boolean = false,
    // Рекорды трассы (картинг-фича). Загружается после слота; при ошибке секция просто скрыта.
    val leaderboard: List<LeaderboardEntry> = emptyList(),
    val waitlist: WaitlistUiState = WaitlistUiState(),
)

/**
 * Лист ожидания на экране заезда. [status] == null — статус не загружен или лист ожидания
 * недоступен: секция не показывается.
 */
data class WaitlistUiState(
    val status: WaitlistStatus? = null,
    val seats: Int = 1,
    val inProgress: Boolean = false,
    val message: String? = null,
)

sealed interface SlotDetailsIntent {
    data class Load(val slotId: SlotId) : SlotDetailsIntent
    data object Retry : SlotDetailsIntent
    data object OpenRouteMap : SlotDetailsIntent
    data object DismissRouteMap : SlotDetailsIntent
    data class SelectWaitlistSeats(val seats: Int) : SlotDetailsIntent
    data object JoinWaitlist : SlotDetailsIntent
    data object LeaveWaitlist : SlotDetailsIntent
    data object Reset : SlotDetailsIntent
}

sealed interface SlotDetailsEffect {
    data object SignedOut : SlotDetailsEffect
}

class SlotDetailsStore(
    private val slotRepository: SlotRepository,
    scope: CoroutineScope? = null,
    private val waitlistRepository: WaitlistRepository = DisabledWaitlistRepository,
) : ViewModel(), MviStore<SlotDetailsState, SlotDetailsIntent, SlotDetailsEffect> {
    private val mutableState = MutableStateFlow(SlotDetailsState())
    private val effects = Channel<SlotDetailsEffect>(Channel.BUFFERED)
    private val storeScope = scope ?: viewModelScope
    private var lastSlotId: SlotId? = null

    override val state: StateFlow<SlotDetailsState> = mutableState

    override fun accept(intent: SlotDetailsIntent) {
        when (intent) {
            is SlotDetailsIntent.Load -> load(intent.slotId)
            SlotDetailsIntent.Retry -> lastSlotId?.let(::load)
            SlotDetailsIntent.OpenRouteMap -> mutableState.update { it.copy(showRouteMap = true) }
            SlotDetailsIntent.DismissRouteMap -> mutableState.update { it.copy(showRouteMap = false) }
            is SlotDetailsIntent.SelectWaitlistSeats -> mutableState.update {
                it.copy(waitlist = it.waitlist.copy(seats = intent.seats.coerceIn(1, MaxWaitlistSeats), message = null))
            }
            SlotDetailsIntent.JoinWaitlist -> joinWaitlist()
            SlotDetailsIntent.LeaveWaitlist -> leaveWaitlist()
            SlotDetailsIntent.Reset -> {
                lastSlotId = null
                mutableState.value = SlotDetailsState()
            }
        }
    }

    override suspend fun effects(): SlotDetailsEffect = effects.receive()

    private fun load(slotId: SlotId) {
        if (mutableState.value.slot == Loadable.Loading && lastSlotId == slotId) return
        lastSlotId = slotId

        storeScope.launch {
            mutableState.update {
                it.copy(slot = Loadable.Loading, showRouteMap = false, leaderboard = emptyList(), waitlist = WaitlistUiState())
            }
            val result = slotRepository.getSlot(slotId)
            // Пока шёл запрос, человек мог открыть другой заезд. На медленной сети ответы
            // приходят в любом порядке, и поздний ответ по старому заезду подменил бы
            // открытый сейчас.
            if (lastSlotId != slotId) return@launch
            result.fold(
                onSuccess = { slot ->
                    mutableState.update { it.copy(slot = Loadable.Content(slot)) }
                    loadLeaderboard(slot)
                    loadWaitlist(slotId)
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to load slot details")
                    val appFailure = failure.asAppFailure()
                    if (appFailure == AppFailure.Unauthorized) {
                        effects.send(SlotDetailsEffect.SignedOut)
                    } else {
                        mutableState.update { it.copy(slot = Loadable.Error(appFailure)) }
                    }
                },
            )
        }
    }

    // Рекорды трассы — вспомогательная секция: ошибку не показываем, просто оставляем пусто.
    private fun loadLeaderboard(slot: Slot) {
        storeScope.launch {
            val result = slotRepository.leaderboard(slot.route.id)
            // Та же гонка: рекорды трассы прошлого заезда не должны оказаться на экране нового.
            if (lastSlotId != slot.id) return@launch
            result.fold(
                onSuccess = { entries -> mutableState.update { it.copy(leaderboard = entries) } },
                onFailure = { failure -> AppLogger.e(failure, "Failed to load leaderboard") },
            )
        }
    }

    // Лист ожидания — тоже вспомогательная секция: не загрузился (или выключен на сервере) —
    // просто не показываем.
    private fun loadWaitlist(slotId: SlotId) {
        storeScope.launch {
            val result = waitlistRepository.status(slotId)
            if (lastSlotId != slotId) return@launch
            result.fold(
                onSuccess = { status -> mutableState.update { it.copy(waitlist = it.waitlist.copy(status = status)) } },
                onFailure = { failure -> AppLogger.e(failure, "Failed to load waitlist status") },
            )
        }
    }

    private fun joinWaitlist() {
        val slotId = lastSlotId ?: return
        val current = mutableState.value.waitlist
        if (current.inProgress || current.status == null) return
        mutableState.update { it.copy(waitlist = it.waitlist.copy(inProgress = true, message = null)) }
        storeScope.launch {
            val result = waitlistRepository.join(slotId, current.seats)
            if (lastSlotId != slotId) return@launch
            result.fold(
                onSuccess = { entry ->
                    mutableState.update {
                        it.copy(waitlist = it.waitlist.copy(
                            status = it.waitlist.status?.copy(entry = entry),
                            inProgress = false,
                        ))
                    }
                },
                onFailure = { failure -> onWaitlistFailure(slotId, failure) },
            )
        }
    }

    private fun leaveWaitlist() {
        val slotId = lastSlotId ?: return
        val current = mutableState.value.waitlist
        if (current.inProgress || current.status?.entry == null) return
        mutableState.update { it.copy(waitlist = it.waitlist.copy(inProgress = true, message = null)) }
        storeScope.launch {
            val result = waitlistRepository.leave(slotId)
            if (lastSlotId != slotId) return@launch
            result.fold(
                onSuccess = {
                    mutableState.update {
                        it.copy(waitlist = it.waitlist.copy(status = it.waitlist.status?.copy(entry = null), inProgress = false))
                    }
                },
                onFailure = { failure -> onWaitlistFailure(slotId, failure) },
            )
        }
    }

    private suspend fun onWaitlistFailure(slotId: SlotId, failure: Throwable) {
        AppLogger.e(failure, "Waitlist request failed")
        val appFailure = failure.asAppFailure()
        if (appFailure == AppFailure.Unauthorized) {
            effects.send(SlotDetailsEffect.SignedOut)
            return
        }
        // Пока человек смотрел на «Мест нет», место освободилось: обновляем заезд — появится
        // кнопка записи.
        if (appFailure is AppFailure.Api && appFailure.code == ApiErrorCode.SeatsAvailable) {
            load(slotId)
            return
        }
        val message = when (appFailure) {
            is AppFailure.Api -> appFailure.message
            AppFailure.NetworkUnavailable, AppFailure.Timeout -> "Нет соединения. Проверьте интернет и попробуйте снова."
            else -> "Не получилось. Попробуйте ещё раз."
        }
        mutableState.update { it.copy(waitlist = it.waitlist.copy(inProgress = false, message = message)) }
    }

    private companion object {
        const val MaxWaitlistSeats = 3
    }
}
