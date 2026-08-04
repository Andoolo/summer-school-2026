package com.volna.app.marshal.presentation

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.volna.app.catalog.presentation.BackButton
import com.volna.app.catalog.presentation.ScreenTitle
import com.volna.app.catalog.presentation.StateArtwork
import com.volna.app.catalog.presentation.StateMessage
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.Slot
import com.volna.app.marshal.RaceParticipant
import com.volna.app.marshal.RaceRoster

/**
 * Режим маршала (F6): внести времена кругов по прошедшему заезду.
 *
 * Экран намеренно отделён от клиентской части приложения — это рабочее место
 * персонала картодрома, а не витрина для гонщика. Доступ даёт токен, а не
 * обычная авторизация по телефону.
 */
@Composable
fun MarshalScreen(
    state: MarshalState,
    onIntent: (MarshalIntent) -> Unit,
    onBack: () -> Unit,
) {
    LaunchedEffect(Unit) {
        onIntent(MarshalIntent.Restore)
    }
    Box(Modifier.fillMaxSize()) {
        BackButton(onBack)
        ScreenTitle("Маршал")

        when {
            state.checkingStoredToken -> Unit
            state.token == null -> TokenGate(state, onIntent)
            state.roster is Loadable.Content -> RosterContent(
                roster = (state.roster as Loadable.Content<RaceRoster>).value,
                state = state,
                onIntent = onIntent,
            )
            else -> RacesContent(state, onIntent)
        }
    }
}

/** Вход в режим: токен вводится один раз и запоминается на устройстве. */
@Composable
private fun TokenGate(state: MarshalState, onIntent: (MarshalIntent) -> Unit) {
    val spacing = VolnaTheme.tokens.spacing
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(top = VolnaTheme.tokens.sizing.listCardTopY)
            .padding(horizontal = spacing.md),
        verticalArrangement = Arrangement.spacedBy(spacing.sm),
    ) {
        Text(
            text = "Рабочее место маршала",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onSurface,
        )
        Text(
            text = "Введите токен, выданный картинг-центром. Он сохранится на этом устройстве, " +
                "вводить каждый раз не придётся.",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        OutlinedTextField(
            value = state.tokenDraft,
            onValueChange = { onIntent(MarshalIntent.TokenDraftChanged(it)) },
            modifier = Modifier.fillMaxWidth(),
            label = { Text("Токен маршала") },
            singleLine = true,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Done),
        )
        state.message?.let { message ->
            Text(
                text = message,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.error,
            )
        }
        Button(
            onClick = { onIntent(MarshalIntent.SubmitToken) },
            modifier = Modifier
                .fillMaxWidth()
                .height(VolnaTheme.tokens.sizing.buttonHeight),
            shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
        ) {
            Text("Войти")
        }
    }
}

/** Список прошедших заездов — только им можно вносить результаты. */
@Composable
private fun RacesContent(state: MarshalState, onIntent: (MarshalIntent) -> Unit) {
    val spacing = VolnaTheme.tokens.spacing
    when (val races = state.races) {
        Loadable.Initial, Loadable.Loading -> Unit

        is Loadable.Empty -> StateMessage(
            title = "Заездов пока нет",
            description = "Результаты можно вносить только по состоявшимся заездам",
            buttonText = "Обновить",
            onClick = { onIntent(MarshalIntent.LoadRaces) },
        )

        is Loadable.Error -> StateMessage(
            title = "Не удалось загрузить",
            description = "Проверьте соединение и попробуйте снова",
            buttonText = "Повторить",
            artwork = StateArtwork.Error,
            onClick = { onIntent(MarshalIntent.LoadRaces) },
        )

        is Loadable.Content -> LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(top = VolnaTheme.tokens.sizing.listCardTopY),
            contentPadding = PaddingValues(
                start = spacing.md,
                end = spacing.md,
                bottom = spacing.xl,
            ),
            verticalArrangement = Arrangement.spacedBy(spacing.sm),
        ) {
            item {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = "Состоявшиеся заезды",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface,
                    )
                    TextButton(onClick = { onIntent(MarshalIntent.SignOut) }) {
                        Text("Выйти")
                    }
                }
            }
            items(races.value) { slot ->
                RaceRow(slot = slot, onClick = { onIntent(MarshalIntent.OpenRace(slot.id)) })
            }
        }
    }
}

@Composable
private fun RaceRow(slot: Slot, onClick: () -> Unit) {
    val spacing = VolnaTheme.tokens.spacing
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(spacing.xl),
            )
            .clickable { onClick() }
            .padding(spacing.md),
        verticalArrangement = Arrangement.spacedBy(spacing.xxs),
    ) {
        Text(
            text = slot.route.name,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onSurface,
        )
        Text(
            text = "${slot.startAt} · маршал ${slot.instructor.name}",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

/** Состав заезда: по каждому гонщику видно, внесены ли круги. */
@Composable
private fun RosterContent(
    roster: RaceRoster,
    state: MarshalState,
    onIntent: (MarshalIntent) -> Unit,
) {
    val spacing = VolnaTheme.tokens.spacing
    LazyColumn(
        modifier = Modifier
            .fillMaxSize()
            .padding(top = VolnaTheme.tokens.sizing.listCardTopY),
        contentPadding = PaddingValues(start = spacing.md, end = spacing.md, bottom = spacing.xl),
        verticalArrangement = Arrangement.spacedBy(spacing.sm),
    ) {
        item {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(Modifier.weight(1f)) {
                    Text(
                        text = roster.routeName,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface,
                    )
                    Text(
                        text = "${roster.participants.size} участников",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                TextButton(onClick = { onIntent(MarshalIntent.CloseRace) }) {
                    Text("К списку")
                }
            }
        }

        state.lastResult?.let { result ->
            item {
                ResultBanner(
                    text = if (result.isTrackRecord) {
                        "Новый рекорд трассы! ${result.racerName} — ${formatLapTime(result.bestLapMs)}"
                    } else {
                        "Сохранено: ${result.racerName}, лучший круг ${formatLapTime(result.bestLapMs)}"
                    },
                    highlight = result.isTrackRecord,
                    onDismiss = { onIntent(MarshalIntent.DismissMessage) },
                )
            }
        }

        items(roster.participants) { participant ->
            ParticipantRow(
                participant = participant,
                isEditing = state.editingBooking == participant.bookingId,
                lapDraft = state.lapDraft,
                submitting = state.submitting,
                message = state.message,
                onIntent = onIntent,
            )
        }
    }
}

@Composable
private fun ParticipantRow(
    participant: RaceParticipant,
    isEditing: Boolean,
    lapDraft: String,
    submitting: Boolean,
    message: String?,
    onIntent: (MarshalIntent) -> Unit,
) {
    val spacing = VolnaTheme.tokens.spacing
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(spacing.xl),
            )
            .padding(spacing.md),
        verticalArrangement = Arrangement.spacedBy(spacing.xs),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Column(Modifier.weight(1f)) {
                Text(
                    text = participant.racerName,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = participant.lapsSummary(),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            if (!isEditing) {
                TextButton(onClick = { onIntent(MarshalIntent.StartEditing(participant.bookingId)) }) {
                    Text(if (participant.laps > 0) "Изменить" else "Внести")
                }
            }
        }

        if (isEditing) {
            OutlinedTextField(
                value = lapDraft,
                onValueChange = { onIntent(MarshalIntent.LapDraftChanged(it)) },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(120.dp),
                label = { Text("Времена кругов") },
                placeholder = { Text("44.120\n42.318\n42.990") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
            )
            Text(
                text = "По кругу в строке. Формат: 41.890 или 1:01.234",
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            message?.let {
                Text(
                    text = it,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.error,
                )
            }
            Row(horizontalArrangement = Arrangement.spacedBy(spacing.xs)) {
                Button(
                    onClick = { onIntent(MarshalIntent.SubmitLaps) },
                    enabled = !submitting,
                    modifier = Modifier.weight(1f),
                    shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
                ) {
                    Text(if (submitting) "Сохраняем…" else "Сохранить")
                }
                TextButton(onClick = { onIntent(MarshalIntent.CancelEditing) }) {
                    Text("Отмена")
                }
            }
        }
    }
}

@Composable
private fun ResultBanner(text: String, highlight: Boolean, onDismiss: () -> Unit) {
    val spacing = VolnaTheme.tokens.spacing
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = if (highlight) {
                    VolnaTheme.tokens.colors.brand
                } else {
                    MaterialTheme.colorScheme.onSurface
                },
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.lg),
            )
            .clickable { onDismiss() }
            .padding(horizontal = spacing.md, vertical = spacing.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = text,
            modifier = Modifier.weight(1f),
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.surface,
            textAlign = TextAlign.Start,
        )
    }
}

private fun RaceParticipant.lapsSummary(): String = when {
    laps == 0 -> "Круги не внесены"
    bestLapMs == null -> "$laps кругов"
    else -> "$laps кругов · лучший ${formatLapTime(bestLapMs)}"
}
