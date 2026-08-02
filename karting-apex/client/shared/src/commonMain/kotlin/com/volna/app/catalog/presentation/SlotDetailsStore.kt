package com.volna.app.catalog.presentation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.volna.app.catalog.SlotRepository
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
)

sealed interface SlotDetailsIntent {
    data class Load(val slotId: SlotId) : SlotDetailsIntent
    data object Retry : SlotDetailsIntent
    data object OpenRouteMap : SlotDetailsIntent
    data object DismissRouteMap : SlotDetailsIntent
    data object Reset : SlotDetailsIntent
}

sealed interface SlotDetailsEffect {
    data object SignedOut : SlotDetailsEffect
}

class SlotDetailsStore(
    private val slotRepository: SlotRepository,
    scope: CoroutineScope? = null,
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
            mutableState.update { it.copy(slot = Loadable.Loading, showRouteMap = false, leaderboard = emptyList()) }
            slotRepository.getSlot(slotId).fold(
                onSuccess = { slot ->
                    mutableState.update { it.copy(slot = Loadable.Content(slot)) }
                    loadLeaderboard(slot)
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
            slotRepository.leaderboard(slot.route.id).fold(
                onSuccess = { entries -> mutableState.update { it.copy(leaderboard = entries) } },
                onFailure = { failure -> AppLogger.e(failure, "Failed to load leaderboard") },
            )
        }
    }
}
