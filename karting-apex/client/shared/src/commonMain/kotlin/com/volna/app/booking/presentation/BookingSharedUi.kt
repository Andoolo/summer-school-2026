package com.volna.app.booking.presentation

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.core.time.AppClock
import com.volna.app.domain.model.Booking
import com.volna.app.domain.model.BookingStatus
import com.volna.app.domain.model.RouteType
import com.volna.app.uikit.icons.Back
import com.volna.app.uikit.icons.Icons
import com.volna.app.uikit.icons.VolnaIcon
import kotlinx.datetime.*
import kotlin.time.Duration.Companion.hours

@Composable
internal fun BookingSkeletonCard(y: androidx.compose.ui.unit.Dp) {
    Box(
        modifier = Modifier
            .width(VolnaTheme.tokens.sizing.contentWidth)
            .height(VolnaTheme.tokens.sizing.listCardHeight)
            .offset(x = VolnaTheme.tokens.spacing.md, y = y)
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
            ),
    )
}

@Composable
internal fun BookingScreenTitle(title: String) {
    Text(
        text = title,
        modifier = Modifier
            .fillMaxWidth()
            .padding(26.dp),
        textAlign = TextAlign.Center,
        style = MaterialTheme.typography.headlineSmall,
        fontWeight = FontWeight.Bold,
    )
}

@Composable
internal fun BookingBackButton(onClick: () -> Unit) {
    Box(
        modifier = Modifier
            .size(40.dp)
            .shadow(4.dp, RoundedCornerShape(200.dp))
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(200.dp))
            .clickable { onClick() },
        contentAlignment = androidx.compose.ui.Alignment.Center,
    ) {
        VolnaIcon(
            imageVector = Icons.Back,
            contentDescription = "Назад",
            tint = MaterialTheme.colorScheme.primary,
            size = 20.dp,
        )
    }
}

@Composable
internal fun BookingStateMessage(
    title: String,
    description: String,
    buttonText: String,
    onClick: () -> Unit,
) {
    Column(
        modifier = Modifier
            .width(VolnaTheme.tokens.sizing.contentWidth)
            .offset(x = VolnaTheme.tokens.spacing.md, y = VolnaTheme.tokens.sizing.listStateMessageY),
        horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xs),
    ) {
        Text(title, style = MaterialTheme.typography.titleMedium, textAlign = TextAlign.Center)
        Text(description, color = MaterialTheme.colorScheme.onSurfaceVariant, textAlign = TextAlign.Center)
        Button(onClick = onClick) {
            Text(buttonText)
        }
    }
}

internal fun Booking.statusLabel(clock: AppClock? = null, pastGroup: Boolean = false): String = when {
    status == BookingStatus.Cancelled -> "Отменена"
    status == BookingStatus.LateCancel -> "Поздняя отмена"
    pastGroup -> "Прошедшая"
    clock != null && slot?.startAt?.let { it <= clock.now() } == true -> "Прошедшая"
    else -> "Активна"
}

internal fun RouteType.toUiText(): String = when (this) {
    RouteType.Novice -> "для новичков"
    RouteType.Experienced -> "для опытных"
}

internal fun RouteType.toTagText(): String = when (this) {
    RouteType.Novice -> "Новичковый"
    RouteType.Experienced -> "Опытный"
}

internal fun cancelDeadlineText(booking: Booking): String =
    "Бесплатно освободить место можно до ${booking.slot?.startAt?.minus(2.hours)?.toUiText() ?: "уточняется"}"

internal fun Instant.toBookingCardStartText(): String {
    val dateTime = toLocalDateTime(TimeZone.currentSystemDefault())
    val weekday = when (dateTime.dayOfWeek) {
        DayOfWeek.MONDAY -> "Пн"
        DayOfWeek.TUESDAY -> "Вт"
        DayOfWeek.WEDNESDAY -> "Ср"
        DayOfWeek.THURSDAY -> "Чт"
        DayOfWeek.FRIDAY -> "Пт"
        DayOfWeek.SATURDAY -> "Сб"
        DayOfWeek.SUNDAY -> "Вс"
    }
    val minute = dateTime.minute.toString().padStart(2, '0')
    return "$weekday, ${dateTime.dayOfMonth} ${dateTime.month.toMonthName()} · ${dateTime.hour}:$minute"
}

private fun Month.toMonthName(): String = when (this) {
    Month.JANUARY -> "января"
    Month.FEBRUARY -> "февраля"
    Month.MARCH -> "марта"
    Month.APRIL -> "апреля"
    Month.MAY -> "мая"
    Month.JUNE -> "июня"
    Month.JULY -> "июля"
    Month.AUGUST -> "августа"
    Month.SEPTEMBER -> "сентября"
    Month.OCTOBER -> "октября"
    Month.NOVEMBER -> "ноября"
    Month.DECEMBER -> "декабря"
}

internal fun Int.pluralPlaces(): String = when {
    this % 10 == 1 && this % 100 != 11 -> "место"
    this % 10 in 2..4 && this % 100 !in 12..14 -> "места"
    else -> "мест"
}

internal fun Int.pluralRentalBoards(): String = when {
    this % 10 == 1 && this % 100 != 11 -> "комплект экипировки"
    this % 10 in 2..4 && this % 100 !in 12..14 -> "комплекта экипировки"
    else -> "комплектов экипировки"
}

internal fun Instant.toUiText(): String =
    toString()
        .replace("T", " ")
        .removeSuffix("Z")
