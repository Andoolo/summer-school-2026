package com.volna.app.catalog.presentation

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.Slot
import com.volna.app.domain.model.SlotId
import com.volna.app.domain.policy.AvailabilityPolicy
import com.volna.app.map.RouteMapSheet
import com.volna.app.uikit.icons.Back
import com.volna.app.uikit.icons.Icons
import com.volna.app.uikit.icons.Share
import com.volna.app.uikit.icons.VolnaIcon

@Composable
fun SlotDetailsScreen(
    slotId: SlotId,
    state: SlotDetailsState,
    onIntent: (SlotDetailsIntent) -> Unit,
    onBack: () -> Unit,
    onBook: (Slot) -> Unit,
) {
    LaunchedEffect(slotId) {
        onIntent(SlotDetailsIntent.Load(slotId))
    }
    Box(Modifier.fillMaxSize()) {
        when (val slot = state.slot) {
            Loadable.Initial,
            Loadable.Loading -> {
                BackButton(onBack)
                ScreenTitle("Заезд")
                SkeletonCard(y = VolnaTheme.tokens.sizing.listCardTopY)
                SkeletonCard(y = VolnaTheme.tokens.sizing.listCardSecondY)
            }
            is Loadable.Content -> SlotDetailsContent(
                slot = slot.value,
                leaderboard = state.leaderboard,
                onBack = onBack,
                onBook = { onBook(slot.value) },
                onOpenMap = { onIntent(SlotDetailsIntent.OpenRouteMap) },
            )
            is Loadable.Empty -> StateMessage(
                title = "Заезд недоступен",
                description = "Попробуйте выбрать другой слот",
                buttonText = "Назад",
                onClick = onBack,
            )
            is Loadable.Error -> StateMessage(
                title = "Не удалось загрузить",
                description = "Проверьте соединение и попробуйте снова",
                buttonText = "Повторить",
                onClick = { onIntent(SlotDetailsIntent.Retry) },
            )
        }
        if (state.showRouteMap) {
            (state.slot as? Loadable.Content)?.value?.let { slot ->
                RouteMapSheet(
                    route = slot.route,
                    meetingPoint = slot.meetingPoint,
                    onDismiss = { onIntent(SlotDetailsIntent.DismissRouteMap) },
                )
            }
        }
    }
}

@Composable
private fun SlotDetailsContent(
    slot: Slot,
    leaderboard: List<com.volna.app.domain.model.LeaderboardEntry>,
    onBack: () -> Unit,
    onBook: () -> Unit,
    onOpenMap: () -> Unit,
) {
    val availability = AvailabilityPolicy.availability(slot)
    Column(Modifier.fillMaxSize()) {
        Box {
            SlotDetailsHero()
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(
                        start = VolnaTheme.tokens.spacing.md,
                        end = VolnaTheme.tokens.spacing.md,
                        top = VolnaTheme.tokens.sizing.backButtonY,
                    ),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                CircleActionButton(icon = Icons.Back, contentDescription = "Назад", onClick = onBack)
                CircleActionButton(icon = Icons.Share, contentDescription = "Поделиться", onClick = {})
            }
        }
        SlotDetailsSheetContent(
            modifier = Modifier
                .fillMaxSize()
                .background(
                    color = MaterialTheme.colorScheme.surface,
                    shape = RoundedCornerShape(
                        topStart = VolnaTheme.tokens.spacing.xl,
                        topEnd = VolnaTheme.tokens.spacing.xl,
                    ),
                ),
            slot = slot,
            availability = availability,
            leaderboard = leaderboard,
            onBook = onBook,
            onOpenMap = onOpenMap,
        )
    }
}

@Composable
private fun SlotDetailsHero() {
    // Картинг-обложка детали заезда: асфальт + красная полоса + клетчатый флаг.
    com.volna.app.uikit.KartingHeroArt(height = 188.dp)
}

@Composable
private fun CircleActionButton(
    icon: ImageVector,
    contentDescription: String,
    onClick: () -> Unit,
) {
    Box(
        modifier = Modifier
            .size(40.dp)
            .shadow(4.dp, RoundedCornerShape(200.dp))
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(200.dp))
            .clickable { onClick() },
        contentAlignment = androidx.compose.ui.Alignment.Center,
    ) {
        VolnaIcon(
            imageVector = icon,
            contentDescription = contentDescription,
            tint = MaterialTheme.colorScheme.primary,
            size = 20.dp,
        )
    }
}

@Composable
private fun SlotDetailsSheetContent(
    slot: Slot,
    availability: com.volna.app.domain.policy.Availability,
    leaderboard: List<com.volna.app.domain.model.LeaderboardEntry>,
    onBook: () -> Unit,
    onOpenMap: () -> Unit,
    modifier: Modifier = Modifier,
) {
    LazyColumn(
        modifier = modifier,
        contentPadding = PaddingValues(bottom = VolnaTheme.tokens.spacing.xs),
        horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
    ) {
        item {
            Box(
                modifier = Modifier
                    .width(40.dp)
                    .height(4.dp)
                    .background(Color(0xFFCCCCCC).copy(alpha = 0.4f), RoundedCornerShape(VolnaTheme.tokens.radius.lg)),
            )
        }
        item {
            Row(horizontalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xxs)) {
                SlotTag(text = slot.route.type.toTagText(), color = Color(0xFF92FF9A))
                SlotTag(
                    text = slot.route.name,
                    color = Color(0xFFFFF897),
                    modifier = Modifier.weight(1f, fill = false),
                )
            }
        }
        item {
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
                Text(
                    text = slot.startAt.toSlotCardStartText(),
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = "Заезд по трассе «${slot.route.name}» займёт ${slot.route.durationMin} минут и отлично подойдёт ${slot.route.type.toDetailsAudienceText()}.",
                    style = MaterialTheme.typography.bodyLarge,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = "Маршал: ${slot.instructor.name}",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
        item {
            SlotDetailsMapCard(slot = slot, onOpenMap = onOpenMap)
        }
        item {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(
                        color = MaterialTheme.colorScheme.surfaceVariant,
                        shape = RoundedCornerShape(VolnaTheme.tokens.spacing.xl),
                    )
                    .padding(VolnaTheme.tokens.spacing.md),
                verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
            ) {
                DetailsInfoRow("Свободно мест", "${slot.freeSeats} из ${slot.totalSeats}")
                DetailsInfoRow(
                    "Экипировка (доступно ${availability.freeRentalBoards} шт.)",
                    "${slot.rentalPrice.value} ₽",
                    boldValue = true
                )
                DetailsInfoRow("Цена", "${slot.price.value} ₽", boldValue = true)
            }
        }
        if (leaderboard.isNotEmpty()) {
            item {
                LeaderboardCard(entries = leaderboard)
            }
        }
        item {
            Text(
                text = "Оплата на месте: наличные или перевод",
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        item {
            Button(
                onClick = onBook,
                enabled = availability.canBook,
                modifier = Modifier
                    .width(VolnaTheme.tokens.sizing.contentWidth)
                    .height(VolnaTheme.tokens.sizing.buttonHeight),
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary),
            ) {
                Text(if (availability.canBook) "Записаться" else "Мест нет", fontWeight = FontWeight.Bold)
            }
        }
        item {
            Box(
                modifier = Modifier
                    .width(138.dp)
                    .height(4.dp)
                    .background(Color(0xFFCCCCCC), RoundedCornerShape(VolnaTheme.tokens.radius.pill)),
            )
        }
    }
}

@Composable
private fun SlotDetailsMapCard(
    slot: Slot,
    onOpenMap: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(VolnaTheme.tokens.spacing.xl),
            )
            .padding(VolnaTheme.tokens.spacing.md),
        horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
    ) {
        Text(
            text = "Адрес: ${slot.meetingPoint.title.ifBlank { "уточняется" }}",
            style = MaterialTheme.typography.bodyMedium,
            textAlign = TextAlign.Center,
            color = MaterialTheme.colorScheme.onSurface,
        )
        SlotDetailsMapPreview()
        Text(
            text = "Открыть карту",
            modifier = Modifier.clickable { onOpenMap() },
            style = MaterialTheme.typography.bodyMedium,
            color = Color(0xFF0093CC),
        )
    }
}

@Composable
private fun SlotDetailsMapPreview() {
    Canvas(
        modifier = Modifier
            .fillMaxWidth()
            .height(156.dp)
            .background(Color.White, RoundedCornerShape(VolnaTheme.tokens.radius.sm)),
    ) {
        val corner = 12.dp.toPx()
        drawRoundRect(
            color = Color(0xFF8AD0F0),
            cornerRadius = CornerRadius(corner, corner),
        )
        drawRoundRect(
            color = Color(0xFFDDF3CC),
            topLeft = Offset(size.width * 0.02f, 0f),
            size = androidx.compose.ui.geometry.Size(size.width * 0.22f, size.height),
            cornerRadius = CornerRadius(corner, corner),
        )
        drawRoundRect(
            color = Color(0xFFDDF3CC),
            topLeft = Offset(size.width * 0.84f, 0f),
            size = androidx.compose.ui.geometry.Size(size.width * 0.16f, size.height),
            cornerRadius = CornerRadius(corner, corner),
        )
        listOf(0.22f, 0.50f, 0.78f).forEach { y ->
            drawLine(
                color = Color(0xFFF9F6F0),
                start = Offset(0f, size.height * y),
                end = Offset(size.width, size.height * (y - 0.12f)),
                strokeWidth = 6.dp.toPx(),
                cap = StrokeCap.Round,
            )
        }
        val routePoints = listOf(
            Offset(size.width * 0.34f, size.height * 0.88f),
            Offset(size.width * 0.48f, size.height * 0.58f),
            Offset(size.width * 0.62f, size.height * 0.36f),
        )
        routePoints.zipWithNext().forEach { (start, end) ->
            drawLine(
                color = Color(0xFF00A59D),
                start = start,
                end = end,
                strokeWidth = 4.dp.toPx(),
                cap = StrokeCap.Round,
            )
        }
        drawCircle(Color(0xFFFF6B4A), radius = 6.dp.toPx(), center = routePoints.first())
        drawCircle(Color.White, radius = 2.5.dp.toPx(), center = routePoints.first())
    }
}

@Composable
private fun DetailsInfoRow(
    label: String,
    value: String,
    boldValue: Boolean = false,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            modifier = Modifier.weight(1f),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface,
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyMedium,
            fontWeight = if (boldValue) FontWeight.Bold else FontWeight.Normal,
            color = MaterialTheme.colorScheme.onSurface,
        )
    }
}

/** Рекорды трассы (картинг-фича): топ лучших кругов по всем прошедшим заездам этой конфигурации. */
@Composable
private fun LeaderboardCard(entries: List<com.volna.app.domain.model.LeaderboardEntry>) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(VolnaTheme.tokens.spacing.xl),
            )
            .padding(VolnaTheme.tokens.spacing.md),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
    ) {
        Text(
            text = "🏁 Рекорды трассы",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onSurface,
        )
        entries.forEach { entry ->
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
            ) {
                Text(
                    text = "${entry.position}",
                    modifier = Modifier.width(24.dp),
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Black,
                    color = if (entry.position == 1) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Text(
                    text = entry.name,
                    modifier = Modifier.weight(1f),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = entry.bestLapMs.toLapTimeText(),
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurface,
                )
            }
        }
        Text(
            text = "Лучший круг по всем заездам конфигурации",
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

/** 41890 → «41.890», 61234 → «1:01.234». */
private fun Int.toLapTimeText(): String {
    val minutes = this / 60_000
    val seconds = (this % 60_000) / 1000
    val millis = (this % 1000).toString().padStart(3, '0')
    return if (minutes > 0) {
        "$minutes:${seconds.toString().padStart(2, '0')}.$millis"
    } else {
        "$seconds.$millis"
    }
}
