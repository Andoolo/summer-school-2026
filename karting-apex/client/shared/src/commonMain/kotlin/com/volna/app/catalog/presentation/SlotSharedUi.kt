package com.volna.app.catalog.presentation

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.domain.model.RouteType
import com.volna.app.uikit.icons.Back
import com.volna.app.uikit.icons.Icons
import com.volna.app.uikit.icons.VolnaIcon
import kotlinx.datetime.DayOfWeek
import kotlinx.datetime.Month
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

@Composable
internal fun SkeletonCard(
    y: androidx.compose.ui.unit.Dp,
    height: androidx.compose.ui.unit.Dp = VolnaTheme.tokens.sizing.listCardHeight,
) {
    Box(
        modifier = Modifier
            .width(VolnaTheme.tokens.sizing.contentWidth)
            .height(height)
            .offset(x = VolnaTheme.tokens.spacing.md, y = y)
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
            ),
    )
}

@Composable
internal fun SlotTag(
    text: String,
    color: Color,
    modifier: Modifier = Modifier,
) {
    Text(
        text = text,
        modifier = modifier
            .background(
                color = color,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.sm),
            )
            .padding(horizontal = VolnaTheme.tokens.spacing.xs, vertical = VolnaTheme.tokens.spacing.xxs),
        style = MaterialTheme.typography.labelMedium,
        color = MaterialTheme.colorScheme.onSurface,
        maxLines = 1,
        overflow = TextOverflow.Ellipsis,
    )
}

@Composable
internal fun BackButton(onClick: () -> Unit) {
    VolnaIcon(
        imageVector = Icons.Back,
        contentDescription = "Назад",
        modifier = Modifier
            .offset(x = VolnaTheme.tokens.spacing.md, y = VolnaTheme.tokens.sizing.backButtonY)
            .clickable { onClick() },
        tint = MaterialTheme.colorScheme.onSurface,
        size = VolnaTheme.tokens.spacing.xl,
    )
}

@Composable
internal fun ScreenTitle(title: String) {
    // Гоночная типографика «Апекс»: сверхжирный курсив с разрядкой.
    Text(
        text = title.uppercase(),
        modifier = Modifier.fillMaxWidth().padding(26.dp),
        textAlign = TextAlign.Center,
        style = MaterialTheme.typography.headlineSmall,
        fontWeight = FontWeight.Black,
        fontStyle = androidx.compose.ui.text.font.FontStyle.Italic,
        letterSpacing = androidx.compose.ui.unit.TextUnit(2f, androidx.compose.ui.unit.TextUnitType.Sp),
    )
}

internal enum class StateArtwork {
    Empty,
    Error,
}

@Composable
internal fun StateMessage(
    title: String,
    description: String,
    buttonText: String? = null,
    artwork: StateArtwork = StateArtwork.Empty,
    onClick: (() -> Unit)? = null,
) {
    Column(
        modifier = Modifier
            .width(VolnaTheme.tokens.sizing.contentWidth)
            .offset(x = VolnaTheme.tokens.spacing.md, y = 190.dp),
        horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xs),
    ) {
        StateIllustration(artwork)
        Spacer(Modifier.height(VolnaTheme.tokens.spacing.md))
        Text(
            text = title,
            modifier = Modifier.fillMaxWidth(),
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold,
            textAlign = TextAlign.Center,
            color = MaterialTheme.colorScheme.onSurface,
        )
        Text(
            text = description,
            modifier = Modifier.fillMaxWidth(),
            style = MaterialTheme.typography.bodyLarge,
            color = Color(0xFF797979),
            textAlign = TextAlign.Center,
        )
        if (buttonText != null && onClick != null) {
            Spacer(Modifier.height(VolnaTheme.tokens.spacing.lg))
            Button(
                onClick = onClick,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(VolnaTheme.tokens.sizing.buttonHeight),
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary),
            ) {
                Text(buttonText)
            }
        }
    }
}

@Composable
private fun StateIllustration(artwork: StateArtwork) {
    val primary = MaterialTheme.colorScheme.primary
    Canvas(
        modifier = Modifier.size(width = 212.dp, height = 150.dp),
    ) {
        val light = Color(0xFFE5FFFC)
        val basePath = androidx.compose.ui.graphics.Path().apply {
            moveTo(size.width * 0.15f, size.height * 0.82f)
            cubicTo(size.width * 0.02f, size.height * 0.46f, size.width * 0.28f, size.height * 0.03f, size.width * 0.52f, size.height * 0.12f)
            cubicTo(size.width * 0.68f, size.height * 0.18f, size.width * 0.72f, size.height * 0.36f, size.width * 0.86f, size.height * 0.34f)
            cubicTo(size.width * 1.04f, size.height * 0.32f, size.width * 1.02f, size.height * 0.84f, size.width * 0.77f, size.height * 0.9f)
            cubicTo(size.width * 0.56f, size.height * 0.96f, size.width * 0.34f, size.height * 0.96f, size.width * 0.15f, size.height * 0.82f)
            close()
        }
        drawPath(basePath, light)
        drawPath(
            path = androidx.compose.ui.graphics.Path().apply {
                moveTo(size.width * 0.05f, size.height * 0.62f)
                cubicTo(size.width * 0.24f, size.height * 0.62f, size.width * 0.27f, size.height * 0.92f, size.width * 0.47f, size.height)
                lineTo(size.width * 0.05f, size.height)
                close()
            },
            color = primary,
        )
        drawPath(
            path = androidx.compose.ui.graphics.Path().apply {
                moveTo(size.width * 0.56f, size.height * 0.88f)
                cubicTo(size.width * 0.7f, size.height * 0.66f, size.width * 0.83f, size.height * 0.62f, size.width * 0.98f, size.height * 0.62f)
                lineTo(size.width * 0.98f, size.height)
                lineTo(size.width * 0.56f, size.height)
                close()
            },
            color = primary,
        )
        if (artwork == StateArtwork.Empty) {
            val cardLeft = size.width * 0.32f
            val cardTop = size.height * 0.4f
            drawRoundRect(
                color = light,
                topLeft = Offset(cardLeft, cardTop),
                size = androidx.compose.ui.geometry.Size(size.width * 0.34f, size.height * 0.42f),
                cornerRadius = CornerRadius(14.dp.toPx(), 14.dp.toPx()),
                style = androidx.compose.ui.graphics.drawscope.Stroke(width = 5.dp.toPx()),
            )
            drawCircle(primary, 4.dp.toPx(), Offset(size.width * 0.44f, size.height * 0.58f))
            drawCircle(primary, 4.dp.toPx(), Offset(size.width * 0.55f, size.height * 0.58f))
            drawArc(
                color = primary,
                startAngle = 205f,
                sweepAngle = 130f,
                useCenter = false,
                topLeft = Offset(size.width * 0.43f, size.height * 0.66f),
                size = androidx.compose.ui.geometry.Size(size.width * 0.15f, size.height * 0.12f),
                style = androidx.compose.ui.graphics.drawscope.Stroke(width = 4.dp.toPx(), cap = StrokeCap.Round),
            )
        } else {
            drawLine(primary, Offset(size.width * 0.22f, size.height * 0.58f), Offset(size.width * 0.5f, size.height * 0.74f), strokeWidth = 3.dp.toPx(), cap = StrokeCap.Round)
            drawLine(primary, Offset(size.width * 0.78f, size.height * 0.58f), Offset(size.width * 0.5f, size.height * 0.74f), strokeWidth = 3.dp.toPx(), cap = StrokeCap.Round)
            drawRoundRect(
                color = primary,
                topLeft = Offset(size.width * 0.24f, size.height * 0.52f),
                size = androidx.compose.ui.geometry.Size(size.width * 0.18f, size.height * 0.16f),
                cornerRadius = CornerRadius(4.dp.toPx(), 4.dp.toPx()),
            )
            drawRoundRect(
                color = primary,
                topLeft = Offset(size.width * 0.62f, size.height * 0.52f),
                size = androidx.compose.ui.geometry.Size(size.width * 0.18f, size.height * 0.16f),
                cornerRadius = CornerRadius(4.dp.toPx(), 4.dp.toPx()),
            )
            drawLine(primary, Offset(size.width * 0.47f, size.height * 0.36f), Offset(size.width * 0.52f, size.height * 0.49f), strokeWidth = 3.dp.toPx())
            drawLine(primary, Offset(size.width * 0.52f, size.height * 0.49f), Offset(size.width * 0.48f, size.height * 0.47f), strokeWidth = 3.dp.toPx())
            drawLine(primary, Offset(size.width * 0.58f, size.height * 0.38f), Offset(size.width * 0.54f, size.height * 0.49f), strokeWidth = 3.dp.toPx())
        }
    }
}

internal fun RouteType.toUiText(): String = when (this) {
    RouteType.Novice -> "для новичков"
    RouteType.Experienced -> "для опытных"
}

internal fun RouteType.toTagText(): String = when (this) {
    RouteType.Novice -> "Новичковый"
    RouteType.Experienced -> "Опытный"
}

internal fun RouteType.toDetailsAudienceText(): String = when (this) {
    RouteType.Novice -> "для новичков"
    RouteType.Experienced -> "для опытных райдеров"
}

internal fun kotlinx.datetime.Instant.toSlotCardStartText(): String {
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
    val month = when (dateTime.month) {
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
    val minute = dateTime.minute.toString().padStart(2, '0')
    return "$weekday, ${dateTime.dayOfMonth} $month · ${dateTime.hour}:$minute"
}

internal fun kotlinx.datetime.Instant?.toFilterDateText(prefix: String, fallback: String): String =
    if (this == null) {
        "$prefix: $fallback"
    } else {
        val date = toLocalDateTime(TimeZone.currentSystemDefault()).date
        "$prefix: ${date.dayOfMonth} ${date.month.toMonthName()}"
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

internal fun kotlinx.datetime.Instant.toUiText(): String =
    toString()
        .replace("T", " ")
        .removeSuffix("Z")
