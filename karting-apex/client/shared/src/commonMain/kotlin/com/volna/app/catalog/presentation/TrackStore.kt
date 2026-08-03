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
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.TrackPassport
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class TrackState(
    val passport: Loadable<TrackPassport> = Loadable.Initial,
    // Рекорды трассы — вспомогательная секция: при ошибке просто остаётся пустой.
    val leaderboard: List<LeaderboardEntry> = emptyList(),
)

sealed interface TrackIntent {
    data class Load(val routeId: RouteId) : TrackIntent
    data object Retry : TrackIntent
    data object Reset : TrackIntent
}

sealed interface TrackEffect {
    data object SignedOut : TrackEffect
}

/** Карточка трассы (F5): паспорт + схема + таблица рекордов. */
class TrackStore(
    private val slotRepository: SlotRepository,
    scope: CoroutineScope? = null,
) : ViewModel(), MviStore<TrackState, TrackIntent, TrackEffect> {
    private val mutableState = MutableStateFlow(TrackState())
    private val effects = Channel<TrackEffect>(Channel.BUFFERED)
    private val storeScope = scope ?: viewModelScope
    private var lastRouteId: RouteId? = null

    override val state: StateFlow<TrackState> = mutableState

    override fun accept(intent: TrackIntent) {
        when (intent) {
            is TrackIntent.Load -> load(intent.routeId)
            TrackIntent.Retry -> lastRouteId?.let(::load)
            TrackIntent.Reset -> {
                lastRouteId = null
                mutableState.value = TrackState()
            }
        }
    }

    override suspend fun effects(): TrackEffect = effects.receive()

    private fun load(routeId: RouteId) {
        if (mutableState.value.passport == Loadable.Loading && lastRouteId == routeId) return
        lastRouteId = routeId

        storeScope.launch {
            mutableState.update { it.copy(passport = Loadable.Loading, leaderboard = emptyList()) }
            slotRepository.trackPassport(routeId).fold(
                onSuccess = { passport ->
                    mutableState.update { it.copy(passport = Loadable.Content(passport)) }
                    loadLeaderboard(routeId)
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to load track passport")
                    val appFailure = failure.asAppFailure()
                    if (appFailure == AppFailure.Unauthorized) {
                        effects.send(TrackEffect.SignedOut)
                    } else {
                        mutableState.update { it.copy(passport = Loadable.Error(appFailure)) }
                    }
                },
            )
        }
    }

    private fun loadLeaderboard(routeId: RouteId) {
        storeScope.launch {
            slotRepository.leaderboard(routeId).fold(
                onSuccess = { entries -> mutableState.update { it.copy(leaderboard = entries) } },
                onFailure = { failure -> AppLogger.e(failure, "Failed to load track leaderboard") },
            )
        }
    }
}
