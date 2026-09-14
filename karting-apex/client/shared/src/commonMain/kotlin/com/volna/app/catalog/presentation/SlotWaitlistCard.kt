package com.volna.app.catalog.presentation

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import com.volna.app.catalog.OfferMinutes
import com.volna.app.catalog.WaitlistEntry
import com.volna.app.catalog.WaitlistEntryStatus
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.core.ui.webSelectionLabel
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

/**
 * Лист ожидания на экране заезда. Показывается, когда мест нет или человек уже в очереди.
 * [canBook] — места есть: встать в очередь нельзя, но запись в очереди (с предложением)
 * всё равно показываем.
 */
@Composable
internal fun SlotWaitlistCard(
    waitlist: WaitlistUiState,
    canBook: Boolean,
    maxSeats: Int,
    onIntent: (SlotDetailsIntent) -> Unit,
    zone: TimeZone = TimeZone.currentSystemDefault(),
) {
    val status = waitlist.status ?: return
    val entry = status.entry
    if (entry == null && canBook) return

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(VolnaTheme.tokens.spacing.xl),
            )
            .padding(VolnaTheme.tokens.spacing.md),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xs),
    ) {
        when {
            entry == null -> JoinContent(
                seats = waitlist.seats,
                maxSeats = maxSeats,
                inProgress = waitlist.inProgress,
                hint = joinBlockedHint(status.telegramLinked, status.notificationsEnabled),
                onIntent = onIntent,
            )
            else -> EntryContent(entry = entry, inProgress = waitlist.inProgress, zone = zone, onIntent = onIntent)
        }
        waitlist.message?.let { message ->
            Text(
                text = message,
                // Ошибку объявляем скринридеру сразу — она появляется после нажатия кнопки.
                modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite },
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.error,
            )
        }
    }
}

@Composable
private fun JoinContent(
    seats: Int,
    maxSeats: Int,
    inProgress: Boolean,
    hint: String?,
    onIntent: (SlotDetailsIntent) -> Unit,
) {
    Title("Лист ожидания")
    Text(
        text = hint ?: ("Мест нет. Встаньте в очередь — когда место освободится, " +
            "пришлём сообщение в Telegram. На запись будет $OfferMinutes минут."),
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurface,
    )
    if (hint != null) return
    Text("Сколько мест нужно", style = MaterialTheme.typography.labelLarge, color = MaterialTheme.colorScheme.onSurfaceVariant)
    Row(horizontalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xs)) {
        for (count in 1..maxSeats.coerceIn(1, 3)) {
            FilterChip(
                selected = count == seats,
                modifier = Modifier.webSelectionLabel(seatsText(count), count == seats),
                // Выбор в цвете темы: по умолчанию Material красит его в сиреневый.
                colors = FilterChipDefaults.filterChipColors(
                    selectedContainerColor = MaterialTheme.colorScheme.primary,
                    selectedLabelColor = MaterialTheme.colorScheme.onPrimary,
                ),
                onClick = { onIntent(SlotDetailsIntent.SelectWaitlistSeats(count)) },
                label = { Text(count.toString()) },
            )
        }
    }
    Button(
        onClick = { onIntent(SlotDetailsIntent.JoinWaitlist) },
        enabled = !inProgress,
        modifier = Modifier.fillMaxWidth().height(VolnaTheme.tokens.sizing.buttonHeight),
        shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
    ) {
        Text(if (inProgress) "Ставим в очередь…" else "Встать в очередь", fontWeight = FontWeight.Bold)
    }
}

@Composable
private fun EntryContent(
    entry: WaitlistEntry,
    inProgress: Boolean,
    zone: TimeZone,
    onIntent: (SlotDetailsIntent) -> Unit,
) {
    when (entry.status) {
        WaitlistEntryStatus.Offered -> {
            Title("Место освободилось!")
            Text(
                text = offerText(entry.offerExpiresAt, zone),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurface,
            )
        }
        WaitlistEntryStatus.Waiting -> {
            Title("Вы в листе ожидания")
            Text(
                text = waitingText(entry.position, entry.seatsCount),
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onSurface,
            )
            Text(
                text = "Когда место освободится, пришлём сообщение в Telegram. На запись будет $OfferMinutes минут.",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
    OutlinedButton(
        onClick = { onIntent(SlotDetailsIntent.LeaveWaitlist) },
        enabled = !inProgress,
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
    ) {
        Text("Выйти из очереди")
    }
}

@Composable
private fun Title(text: String) {
    Text(
        text = text,
        modifier = Modifier.semantics { heading() },
        style = MaterialTheme.typography.titleMedium,
        fontWeight = FontWeight.Bold,
        color = MaterialTheme.colorScheme.onSurface,
    )
}

internal fun joinBlockedHint(telegramLinked: Boolean, notificationsEnabled: Boolean): String? = when {
    !telegramLinked -> "Мест нет. О свободном месте мы сообщаем в Telegram — чтобы встать в очередь, " +
        "выйдите из профиля и войдите через Telegram."
    !notificationsEnabled -> "Мест нет. Уведомления в Telegram отключены — отправьте боту /notify, " +
        "и сможете встать в очередь."
    else -> null
}

internal fun waitingText(position: Int, seats: Int): String {
    val place = if (position > 0) "$position-й в очереди" else "в очереди"
    return "$place · ${seatsText(seats)}"
}

internal fun offerText(expiresAt: Instant?, zone: TimeZone): String {
    val until = expiresAt?.toLocalDateTime(zone)?.let { " до ${it.hour}:${it.minute.toString().padStart(2, '0')}" }.orEmpty()
    return "Запишитесь$until — потом предложим место следующему в очереди. " +
        "Место не закреплено: пока вы не записались, его может занять другой."
}

private fun seatsText(seats: Int): String = when (seats) {
    1 -> "1 место"
    else -> "$seats места"
}
