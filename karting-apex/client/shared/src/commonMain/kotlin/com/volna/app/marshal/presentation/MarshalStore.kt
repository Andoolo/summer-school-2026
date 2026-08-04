package com.volna.app.marshal.presentation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.volna.app.catalog.PageRequest
import com.volna.app.catalog.SlotFilters
import com.volna.app.catalog.SlotRepository
import com.volna.app.core.error.ApiErrorCode
import com.volna.app.core.error.AppFailure
import com.volna.app.core.error.asAppFailure
import com.volna.app.core.logging.AppLogger
import com.volna.app.core.mvi.MviStore
import com.volna.app.core.storage.MarshalTokenStorage
import com.volna.app.core.time.AppClock
import com.volna.app.core.ui.EmptyReason
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.BookingId
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.marshal.LapEntryResult
import com.volna.app.marshal.MarshalRepository
import com.volna.app.marshal.RaceRoster
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.channels.Channel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch

data class MarshalState(
    /** null — токен ещё не введён: показываем вход в режим маршала. */
    val token: String? = null,
    val tokenDraft: String = "",
    val checkingStoredToken: Boolean = true,
    /** Прошедшие заезды: только им можно вносить результаты. */
    val races: Loadable<List<Slot>> = Loadable.Initial,
    val roster: Loadable<RaceRoster> = Loadable.Initial,
    /** Бронь, для которой открыт ввод времён. */
    val editingBooking: BookingId? = null,
    val lapDraft: String = "",
    val submitting: Boolean = false,
    val message: String? = null,
    val lastResult: LapEntryResult? = null,
)

sealed interface MarshalIntent {
    data object Restore : MarshalIntent
    data class TokenDraftChanged(val value: String) : MarshalIntent
    data object SubmitToken : MarshalIntent
    data object SignOut : MarshalIntent
    data object LoadRaces : MarshalIntent
    data class OpenRace(val slotId: SlotId) : MarshalIntent
    data object CloseRace : MarshalIntent
    data class StartEditing(val bookingId: BookingId) : MarshalIntent
    data class LapDraftChanged(val value: String) : MarshalIntent
    data object CancelEditing : MarshalIntent
    data object SubmitLaps : MarshalIntent
    data object DismissMessage : MarshalIntent
    data object Reset : MarshalIntent
}

sealed interface MarshalEffect {
    data object TokenRejected : MarshalEffect
}

/**
 * Режим маршала (F6): ввод времён кругов по прошедшим заездам.
 *
 * Токен хранится отдельно от клиентской сессии и переживает выход гонщика из
 * аккаунта — это настройка рабочего места, а не вход пользователя.
 */
class MarshalStore(
    private val slotRepository: SlotRepository,
    private val marshalRepository: MarshalRepository,
    private val tokenStorage: MarshalTokenStorage,
    private val clock: AppClock,
    scope: CoroutineScope? = null,
) : ViewModel(), MviStore<MarshalState, MarshalIntent, MarshalEffect> {
    private val mutableState = MutableStateFlow(MarshalState())
    private val effects = Channel<MarshalEffect>(Channel.BUFFERED)
    private val storeScope = scope ?: viewModelScope

    override val state: StateFlow<MarshalState> = mutableState

    override fun accept(intent: MarshalIntent) {
        when (intent) {
            MarshalIntent.Restore -> restore()
            is MarshalIntent.TokenDraftChanged ->
                mutableState.update { it.copy(tokenDraft = intent.value, message = null) }
            MarshalIntent.SubmitToken -> submitToken()
            MarshalIntent.SignOut -> signOut()
            MarshalIntent.LoadRaces -> loadRaces()
            is MarshalIntent.OpenRace -> openRace(intent.slotId)
            MarshalIntent.CloseRace ->
                mutableState.update {
                    it.copy(roster = Loadable.Initial, editingBooking = null, lapDraft = "", lastResult = null)
                }
            is MarshalIntent.StartEditing ->
                mutableState.update { it.copy(editingBooking = intent.bookingId, lapDraft = "", message = null) }
            is MarshalIntent.LapDraftChanged ->
                mutableState.update { it.copy(lapDraft = intent.value, message = null) }
            MarshalIntent.CancelEditing ->
                mutableState.update { it.copy(editingBooking = null, lapDraft = "") }
            MarshalIntent.SubmitLaps -> submitLaps()
            MarshalIntent.DismissMessage -> mutableState.update { it.copy(message = null, lastResult = null) }
            MarshalIntent.Reset -> mutableState.value = MarshalState(checkingStoredToken = false)
        }
    }

    override suspend fun effects(): MarshalEffect = effects.receive()

    private fun restore() {
        storeScope.launch {
            val stored = tokenStorage.readToken()
            mutableState.update { it.copy(token = stored, checkingStoredToken = false) }
            if (stored != null) loadRaces()
        }
    }

    private fun submitToken() {
        val candidate = mutableState.value.tokenDraft.trim()
        if (candidate.isEmpty()) {
            mutableState.update { it.copy(message = "Введите токен маршала") }
            return
        }
        storeScope.launch {
            tokenStorage.writeToken(candidate)
            mutableState.update { it.copy(token = candidate, tokenDraft = "", message = null) }
            loadRaces()
        }
    }

    private fun signOut() {
        storeScope.launch {
            tokenStorage.clearToken()
            mutableState.value = MarshalState(checkingStoredToken = false)
        }
    }

    /**
     * Список заездов, которые уже состоялись. Фильтр по дате делает бэкенд:
     * тянуть весь каталог и отсеивать на клиенте — лишний трафик и риск
     * пропустить заезд из-за пагинации.
     */
    private fun loadRaces() {
        storeScope.launch {
            mutableState.update { it.copy(races = Loadable.Loading) }
            slotRepository.listSlots(
                filters = SlotFilters(dateTo = clock.now()),
                page = PageRequest(limit = 50, offset = 0),
            ).fold(
                onSuccess = { page ->
                    val past = page.items.sortedByDescending { it.startAt }
                    mutableState.update {
                        it.copy(
                            races = if (past.isEmpty()) {
                                Loadable.Empty(EmptyReason.NoSlots)
                            } else {
                                Loadable.Content(past)
                            },
                        )
                    }
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to load past races")
                    mutableState.update { it.copy(races = Loadable.Error(failure.asAppFailure())) }
                },
            )
        }
    }

    private fun openRace(slotId: SlotId) {
        val token = mutableState.value.token ?: return
        storeScope.launch {
            mutableState.update { it.copy(roster = Loadable.Loading, editingBooking = null, lastResult = null) }
            marshalRepository.raceRoster(token, slotId).fold(
                onSuccess = { roster -> mutableState.update { it.copy(roster = Loadable.Content(roster)) } },
                onFailure = { failure -> handleFailure(failure) { mutableState.update { it.copy(roster = Loadable.Error(failure.asAppFailure())) } } },
            )
        }
    }

    private fun submitLaps() {
        val current = mutableState.value
        val token = current.token ?: return
        val bookingId = current.editingBooking ?: return

        val laps = current.lapDraft
            .split('\n', ' ', ';')
            .map { it.trim() }
            .filter { it.isNotEmpty() }
            .map { parseLapTime(it) }

        if (laps.isEmpty()) {
            mutableState.update { it.copy(message = "Введите хотя бы один круг") }
            return
        }
        if (laps.any { it == null }) {
            mutableState.update { it.copy(message = "Время круга: 41.890 или 1:01.234") }
            return
        }

        storeScope.launch {
            mutableState.update { it.copy(submitting = true, message = null) }
            marshalRepository.submitLapResults(token, bookingId, laps.filterNotNull()).fold(
                onSuccess = { result ->
                    mutableState.update {
                        it.copy(submitting = false, editingBooking = null, lapDraft = "", lastResult = result)
                    }
                    // Перечитываем состав, чтобы в списке сразу были новые круги.
                    (mutableState.value.roster as? Loadable.Content)?.value?.let { openRace(it.slotId) }
                },
                onFailure = { failure ->
                    AppLogger.e(failure, "Failed to submit lap results")
                    handleFailure(failure) {
                        mutableState.update {
                            it.copy(submitting = false, message = "Не удалось сохранить результаты")
                        }
                    }
                },
            )
        }
    }

    /**
     * Отказ по токену обрабатываем отдельно: маршал должен понять, что токен
     * неверный или отозван, а не смотреть на общую ошибку сети. Заодно чистим
     * сохранённый токен — держать заведомо нерабочий незачем.
     */
    private suspend fun handleFailure(failure: Throwable, otherwise: () -> Unit) {
        val appFailure = failure.asAppFailure()
        val forbidden = appFailure is AppFailure.Api && appFailure.code == ApiErrorCode.Forbidden
        if (forbidden) {
            tokenStorage.clearToken()
            mutableState.value = MarshalState(checkingStoredToken = false, message = "Токен не принят")
            effects.send(MarshalEffect.TokenRejected)
            return
        }
        otherwise()
    }
}
