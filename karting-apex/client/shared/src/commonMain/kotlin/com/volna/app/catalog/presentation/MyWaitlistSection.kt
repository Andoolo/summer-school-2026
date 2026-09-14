package com.volna.app.catalog.presentation

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import com.volna.app.catalog.MyWaitlistEntry
import com.volna.app.catalog.WaitlistEntryStatus
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.domain.model.SlotId
import kotlinx.datetime.Instant
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

/** Раздел «Мои очереди» в профиле. Пустой список — раздела нет. */
@Composable
fun MyWaitlistSection(
    state: MyWaitlistState,
    onIntent: (MyWaitlistIntent) -> Unit,
    onOpenSlot: (SlotId) -> Unit,
    zone: TimeZone = TimeZone.currentSystemDefault(),
) {
    if (state.entries.isEmpty() && state.message == null) return
    Column(verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xs)) {
        Text(
            text = "Мои очереди",
            modifier = Modifier.semantics { heading() },
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onSurface,
        )
        state.entries.forEach { entry ->
            MyWaitlistRow(
                entry = entry,
                leaving = entry.slotId in state.leaving,
                zone = zone,
                onOpen = { onOpenSlot(entry.slotId) },
                onLeave = { onIntent(MyWaitlistIntent.Leave(entry.slotId)) },
            )
        }
        state.message?.let { message ->
            Text(
                text = message,
                modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite },
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.error,
            )
        }
    }
}

@Composable
private fun MyWaitlistRow(
    entry: MyWaitlistEntry,
    leaving: Boolean,
    zone: TimeZone,
    onOpen: () -> Unit,
    onLeave: () -> Unit,
) {
    val offered = entry.entry.status == WaitlistEntryStatus.Offered
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(MaterialTheme.colorScheme.surfaceVariant, RoundedCornerShape(VolnaTheme.tokens.radius.lg))
            .clickable(onClickLabel = "Открыть заезд", role = Role.Button, onClick = onOpen)
            .padding(start = VolnaTheme.tokens.spacing.md, top = VolnaTheme.tokens.spacing.sm, bottom = VolnaTheme.tokens.spacing.sm),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xxs)) {
            Text(
                text = entry.routeName,
                style = MaterialTheme.typography.bodyLarge,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onSurface,
            )
            Text(
                text = entry.startAt.toSlotCardStartText(zone),
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
            Text(
                text = myEntryStatusText(entry, zone),
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = if (offered) FontWeight.Bold else FontWeight.Normal,
                color = if (offered) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurface,
            )
        }
        TextButton(
            onClick = onLeave,
            enabled = !leaving,
            // Кнопок «Выйти» несколько — скринридеру нужно сказать, из какой очереди.
            modifier = Modifier.semantics { contentDescription = "Выйти из очереди: ${entry.routeName}" },
        ) {
            Text(if (leaving) "Выходим…" else "Выйти")
        }
    }
}

/** «3-й в очереди · 1 место» или «Место освободилось — запишитесь до 18:24». */
internal fun myEntryStatusText(entry: MyWaitlistEntry, zone: TimeZone): String = when (entry.entry.status) {
    WaitlistEntryStatus.Waiting -> waitingText(entry.entry.position, entry.entry.seatsCount)
    WaitlistEntryStatus.Offered -> {
        val until = entry.entry.offerExpiresAt?.let { " до " + it.toClockText(zone) }.orEmpty()
        "Место освободилось — запишитесь$until"
    }
}

private fun Instant.toClockText(zone: TimeZone): String {
    val local = toLocalDateTime(zone)
    return "${local.hour}:${local.minute.toString().padStart(2, '0')}"
}
