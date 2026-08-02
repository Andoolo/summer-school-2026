package com.volna.app.booking.presentation

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.Booking
import com.volna.app.domain.model.BookingId
import com.volna.app.domain.policy.BookingPriceCalculator
import kotlinx.coroutines.delay

@Composable
fun BookingListScreen(
    state: BookingListState,
    onIntent: (BookingListIntent) -> Unit,
    onBookingClick: (BookingId) -> Unit,
    onBookWalk: () -> Unit,
) {
    LaunchedEffect(Unit) {
        onIntent(BookingListIntent.Load)
    }
    LaunchedEffect(state.message) {
        if (state.message != null) {
            delay(2_500)
            onIntent(BookingListIntent.MessageShown)
        }
    }
    Box(Modifier.fillMaxSize()) {
        BookingScreenTitle("Мои записи")
        when (val bookings = state.bookings) {
            Loadable.Initial,
            Loadable.Loading -> {
                BookingSkeletonCard(y = VolnaTheme.tokens.sizing.listCardTopY)
                BookingSkeletonCard(y = VolnaTheme.tokens.sizing.listCardSecondY)
            }
            is Loadable.Content -> BookingGroupsContent(
                groups = bookings.value,
                refreshing = bookings.refreshing,
                message = state.message,
                onBookingClick = onBookingClick,
                onBookWalk = onBookWalk,
            )
            is Loadable.Empty -> BookingStateMessage(
                title = "У вас пока нет записей",
                description = "Выберите заезд и оформите бронь",
                buttonText = "Записаться",
                onClick = onBookWalk,
            )
            is Loadable.Error -> BookingStateMessage(
                title = "Не удалось загрузить записи",
                description = "Проверьте соединение и попробуйте снова",
                buttonText = "Обновить",
                onClick = { onIntent(BookingListIntent.Retry) },
            )
        }
    }
}

@Composable
private fun BookingGroupsContent(
    groups: BookingGroups,
    refreshing: Boolean,
    message: String?,
    onBookingClick: (BookingId) -> Unit,
    onBookWalk: () -> Unit,
) {
    var selectedTab by remember { mutableStateOf(BookingListTab.Upcoming) }
    val visibleBookings = when (selectedTab) {
        BookingListTab.Upcoming -> groups.upcoming
        BookingListTab.Past -> groups.past
    }
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.TopCenter,
    ) {
        LazyColumn(
            modifier = Modifier
                .width(VolnaTheme.tokens.sizing.contentWidth)
                .fillMaxHeight(),
            contentPadding = PaddingValues(
                top = VolnaTheme.tokens.sizing.listCardTopY,
                bottom = VolnaTheme.tokens.sizing.navContentBottomPadding + VolnaTheme.tokens.spacing.lg,
            ),
            verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.md),
        ) {
            if (refreshing) {
                item {
                    Text("Обновляем записи...", color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
            message?.let {
                item {
                    Text(it, color = MaterialTheme.colorScheme.error)
                }
            }
            item {
                BookingTabs(
                    selected = selectedTab,
                    onSelected = { selectedTab = it },
                )
            }
            if (visibleBookings.isEmpty()) {
                item {
                    BookingEmptyCard(
                        title = if (selectedTab == BookingListTab.Upcoming) {
                            "Пока нет предстоящих записей"
                        } else {
                            "Здесь появятся прошедшие заезды"
                        },
                        description = if (selectedTab == BookingListTab.Upcoming) {
                            "Можно выбрать ближайший заезд"
                        } else {
                            "Отменённые записи тоже будут здесь"
                        },
                        onBookWalk = onBookWalk,
                    )
                }
            } else {
                items(visibleBookings, key = { it.id.value }) { booking ->
                    BookingCard(
                        booking = booking,
                        pastGroup = selectedTab == BookingListTab.Past,
                        onClick = { onBookingClick(booking.id) },
                    )
                }
            }
        }
    }
}

private enum class BookingListTab {
    Upcoming,
    Past,
}

@Composable
private fun BookingTabs(
    selected: BookingListTab,
    onSelected: (BookingListTab) -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(40.dp)
            .border(1.dp, MaterialTheme.colorScheme.primary, RoundedCornerShape(VolnaTheme.tokens.radius.pill))
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(VolnaTheme.tokens.radius.pill)),
        verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
    ) {
        BookingTabButton(
            text = "Предстоящие",
            selected = selected == BookingListTab.Upcoming,
            onClick = { onSelected(BookingListTab.Upcoming) },
        )
        BookingTabButton(
            text = "Прошедшие",
            selected = selected == BookingListTab.Past,
            onClick = { onSelected(BookingListTab.Past) },
        )
    }
}

@Composable
private fun BookingTabButton(
    text: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Text(
        text = text,
        modifier = Modifier
            .width(180.dp)
            .height(40.dp)
            .background(
                color = if (selected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surface,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
            )
            .clickable { onClick() }
            .padding(top = 10.dp),
        textAlign = TextAlign.Center,
        style = MaterialTheme.typography.bodyMedium,
        color = if (selected) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurface,
    )
}

@Composable
private fun BookingEmptyCard(
    title: String,
    description: String,
    onBookWalk: () -> Unit,
) {
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
        Text(title, fontWeight = FontWeight.Bold)
        Text(description, color = MaterialTheme.colorScheme.onSurfaceVariant)
        OutlinedButton(onClick = onBookWalk, modifier = Modifier.fillMaxWidth()) {
            Text("Записаться")
        }
    }
}

@Composable
private fun BookingCard(
    booking: Booking,
    pastGroup: Boolean,
    onClick: () -> Unit,
) {
    val slot = booking.slot
    val status = booking.statusLabel(pastGroup = pastGroup)
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() }
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(VolnaTheme.tokens.spacing.xl),
            )
            .padding(VolnaTheme.tokens.spacing.md),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
    ) {
        BookingPreviewPhoto()
        Column(verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xxs)) {
            Row(horizontalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xxs)) {
                slot?.let {
                    BookingTag(text = it.route.type.toTagText(), color = Color(0xFF92FF9A))
                    BookingTag(
                        text = it.route.name,
                        color = Color(0xFFFFF897),
                        modifier = Modifier.weight(1f, fill = false),
                    )
                }
            }
            Text(
                text = slot?.startAt?.toBookingCardStartText() ?: "Время уточняется",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onSurface,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
        }
        Text(
            text = "Маршал: ${slot?.instructor?.name ?: "уточняется"}",
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurface,
        )
        Row(
            modifier = Modifier
                .fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
        ) {
            Text(
                text = "${booking.seatsCount} ${booking.seatsCount.pluralPlaces()} · ${booking.rentalCount} ${booking.rentalCount.pluralRentalBoards()}",
                modifier = Modifier.weight(1f),
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurface,
            )
            Text(
                text = "${BookingPriceCalculator.calculate(booking)?.value ?: 0} ₽",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onSurface,
            )
        }
        BookingStatusBadge(status)
    }
}

@Composable
internal fun BookingPreviewPhoto() {
    // Картинг-обложка: асфальт + красная полоса + клетчатый флаг.
    com.volna.app.uikit.KartingHeroArt(
        height = 120.dp,
        shape = RoundedCornerShape(VolnaTheme.tokens.radius.lg),
    )
}

@Composable
internal fun BookingTag(
    text: String,
    color: Color,
    modifier: Modifier = Modifier,
) {
    Text(
        text = text,
        modifier = modifier
            .background(color, RoundedCornerShape(VolnaTheme.tokens.radius.sm))
            .padding(horizontal = VolnaTheme.tokens.spacing.xs, vertical = VolnaTheme.tokens.spacing.xxs),
        style = MaterialTheme.typography.labelMedium,
        color = MaterialTheme.colorScheme.onSurface,
        maxLines = 1,
        overflow = TextOverflow.Ellipsis,
    )
}

@Composable
private fun BookingStatusBadge(status: String) {
    val active = status == "Активна"
    Text(
        text = status,
        modifier = Modifier
            .fillMaxWidth()
            .height(36.dp)
            .background(
                color = if (active) Color(0xFFE4FFE5) else MaterialTheme.colorScheme.surface,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.lg),
            )
            .padding(top = 9.dp),
        textAlign = TextAlign.Center,
        style = MaterialTheme.typography.bodyMedium,
        color = if (active) Color(0xFF007108) else MaterialTheme.colorScheme.onSurfaceVariant,
    )
}
