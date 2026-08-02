package com.volna.app.booking.presentation

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.core.time.AppClock
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.Booking
import com.volna.app.domain.model.BookingId
import com.volna.app.domain.policy.BookingPriceCalculator
import com.volna.app.domain.policy.CancellationKind
import com.volna.app.map.RouteMapSheet
import com.volna.app.uikit.icons.Icons
import com.volna.app.uikit.icons.Info
import com.volna.app.uikit.icons.VolnaIcon

// CMP-12 / SCR-006 / BS-003: booking details with explicit cancel confirmation.
@Composable
fun BookingDetailsScreen(
    bookingId: BookingId,
    state: BookingDetailsState,
    clock: AppClock,
    onIntent: (BookingDetailsIntent) -> Unit,
    onBack: () -> Unit,
) {
    LaunchedEffect(bookingId) {
        onIntent(BookingDetailsIntent.Load(bookingId))
    }
    Column(modifier = Modifier.fillMaxSize()) {
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = VolnaTheme.tokens.spacing.md, vertical = 18.dp),
        ) {
            BookingBackButton(onBack)
            BookingScreenTitle("Детали записи")
        }
        Box(modifier = Modifier.fillMaxSize()) {
            when (val booking = state.booking) {
                Loadable.Initial,
                Loadable.Loading -> {
                    BookingSkeletonCard(y = VolnaTheme.tokens.sizing.listCardTopY)
                    BookingSkeletonCard(y = VolnaTheme.tokens.sizing.listCardSecondY)
                }

                is Loadable.Content -> BookingDetailsContent(
                    booking = booking.value,
                    state = state,
                    clock = clock,
                    onIntent = onIntent,
                )

                is Loadable.Empty -> BookingStateMessage(
                    title = "Запись недоступна",
                    description = "Вернитесь к списку и попробуйте снова",
                    buttonText = "Назад",
                    onClick = onBack,
                )

                is Loadable.Error -> BookingStateMessage(
                    title = "Не удалось загрузить запись",
                    description = "Проверьте соединение и попробуйте снова",
                    buttonText = "Обновить",
                    onClick = { onIntent(BookingDetailsIntent.Retry) },
                )
            }
            if (state.showCancelConfirm) {
                CancelConfirmSheet(
                    state = state,
                    clock = clock,
                    onIntent = onIntent,
                )
            }
            if (state.showRouteMap) {
                state.currentBooking?.slot?.let { slot ->
                    RouteMapSheet(
                        route = slot.route,
                        meetingPoint = slot.meetingPoint,
                        onDismiss = { onIntent(BookingDetailsIntent.DismissRouteMap) },
                    )
                }
            }
        }
    }
}

@Composable
private fun BookingDetailsContent(
    booking: Booking,
    state: BookingDetailsState,
    clock: AppClock,
    onIntent: (BookingDetailsIntent) -> Unit,
) {
    val slot = booking.slot
    val canCancel = state.canCancel(clock)
    Column(
        modifier = Modifier
            .width(VolnaTheme.tokens.sizing.contentWidth)
            .padding(start = VolnaTheme.tokens.spacing.md)
            .verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
    ) {
        BookingDetailsEventCard(
            booking = booking,
            status = booking.statusLabel(clock),
        )
        slot?.let {
            BookingDetailsMapCard(
                address = it.meetingPoint.title.ifBlank { "уточняется" },
                onOpenMap = { onIntent(BookingDetailsIntent.OpenRouteMap) },
            )
        }
        BookingDetailsPriceBlock(booking)
        if (canCancel) {
            Text(
                text = cancelDeadlineText(booking),
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        booking.cancelledAt?.let {
            Text(
                text = "Отменено: ${it.toUiText()}",
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
        state.message?.let { Text(it, color = MaterialTheme.colorScheme.primary) }
        Button(
            onClick = { onIntent(BookingDetailsIntent.AskCancel) },
            enabled = canCancel,
            modifier = Modifier
                .fillMaxWidth()
                .height(VolnaTheme.tokens.sizing.buttonHeight),
            shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
        ) {
            Text(if (canCancel) "Отменить" else "Отмена недоступна")
        }
        Box(
            modifier = Modifier
                .width(138.dp)
                .height(4.dp)
                .align(androidx.compose.ui.Alignment.CenterHorizontally)
                .background(Color(0xFFCCCCCC), RoundedCornerShape(VolnaTheme.tokens.radius.pill)),
        )
        Spacer(Modifier.height(VolnaTheme.tokens.spacing.xs))
    }
}

@Composable
private fun BookingDetailsEventCard(
    booking: Booking,
    status: String,
) {
    val slot = booking.slot
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
        Box {
            BookingPreviewPhoto()
            BookingStatusPill(
                status = status,
                modifier = Modifier
                    .offset(x = VolnaTheme.tokens.spacing.xs, y = VolnaTheme.tokens.spacing.xs),
            )
        }
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
        )
        Text(
            text = "Маршал: ${slot?.instructor?.name ?: "уточняется"}",
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurface,
        )
    }
}

@Composable
private fun BookingStatusPill(
    status: String,
    modifier: Modifier = Modifier,
) {
    val active = status == "Активна"
    Text(
        text = status,
        modifier = modifier
            .width(100.dp)
            .height(36.dp)
            .background(
                color = if (active) Color(0xFFE4FFE5) else MaterialTheme.colorScheme.surface,
                shape = RoundedCornerShape(10.dp),
            )
            .padding(top = 9.dp),
        textAlign = TextAlign.Center,
        style = MaterialTheme.typography.bodyMedium,
        color = if (active) Color(0xFF007108) else MaterialTheme.colorScheme.onSurfaceVariant,
    )
}

@Composable
private fun BookingDetailsMapCard(
    address: String,
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
            text = "Адрес: $address",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface,
            textAlign = TextAlign.Center,
        )
        BookingDetailsMapPreview()
        Text(
            text = "Открыть карту",
            modifier = Modifier.clickable { onOpenMap() },
            style = MaterialTheme.typography.bodyMedium,
            color = Color(0xFF0093CC),
        )
    }
}

@Composable
private fun BookingDetailsMapPreview() {
    Canvas(
        modifier = Modifier
            .fillMaxWidth()
            .height(156.dp)
            .background(Color.White, RoundedCornerShape(VolnaTheme.tokens.radius.sm)),
    ) {
        val corner = 12.dp.toPx()
        drawRoundRect(Color(0xFF8AD0F0), cornerRadius = CornerRadius(corner, corner))
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
            drawLine(Color(0xFF00A59D), start, end, strokeWidth = 4.dp.toPx(), cap = StrokeCap.Round)
        }
        drawCircle(Color(0xFFFF6B4A), radius = 6.dp.toPx(), center = routePoints.first())
        drawCircle(Color.White, radius = 2.5.dp.toPx(), center = routePoints.first())
    }
}

@Composable
private fun BookingDetailsPriceBlock(booking: Booking) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .border(width = 1.dp, color = MaterialTheme.colorScheme.surfaceVariant, shape = RoundedCornerShape(0.dp))
            .padding(top = VolnaTheme.tokens.spacing.sm),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xs),
    ) {
        Row(
            modifier = Modifier.fillMaxWidth(),
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
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xxs),
            verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
        ) {
            VolnaIcon(
                imageVector = Icons.Info,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                size = 16.dp,
            )
            Text(
                text = "Оплата на месте: наличные или перевод",
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

@Composable
private fun BookingInfoBlock(content: @Composable ColumnScope.() -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.lg),
            )
            .padding(VolnaTheme.tokens.spacing.md),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xs),
        content = content,
    )
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CancelConfirmSheet(
    state: BookingDetailsState,
    clock: AppClock,
    onIntent: (BookingDetailsIntent) -> Unit,
) {
    val kind = state.cancellationKind(clock)
    val messageText = when (kind) {
        CancellationKind.Early -> "До старта больше 2 часов. Запись будет отменена, места и экипировка снова станут доступны."
        CancellationKind.Late -> "До старта осталось менее 2 часов. Запись будет отменена. Штраф за позднюю отмену не взимается."
        CancellationKind.UnavailableAfterStart,
        null -> "Заезд уже начался. Отмена записи недоступна."
    }
    val cancellationLabel = when (kind) {
        CancellationKind.Early -> "Ранняя отмена"
        CancellationKind.Late -> "Поздняя отмена"
        CancellationKind.UnavailableAfterStart,
        null -> "Отмена недоступна"
    }
    val cancellationHint = when (kind) {
        CancellationKind.Early -> "Места и экипировка освободятся."
        CancellationKind.Late -> "Ваша запись отменена. Штраф не взимается."
        CancellationKind.UnavailableAfterStart,
        null -> "Запись останется активной."
    }
    val sheetState = rememberModalBottomSheetState(
        skipPartiallyExpanded = true,
        confirmValueChange = { !state.isCancelling },
    )

    ModalBottomSheet(
        onDismissRequest = {
            if (!state.isCancelling) {
                onIntent(BookingDetailsIntent.DismissCancel)
            }
        },
        sheetState = sheetState,
        shape = RoundedCornerShape(
            topStart = VolnaTheme.tokens.radius.lg,
            topEnd = VolnaTheme.tokens.radius.lg,
        ),
        containerColor = MaterialTheme.colorScheme.surface,
        dragHandle = {
            Box(
                modifier = Modifier
                    .padding(top = VolnaTheme.tokens.spacing.xs)
                    .size(width = 40.dp, height = 4.dp)
                    .background(
                        color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f),
                        shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
                    ),
            )
        },
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .verticalScroll(rememberScrollState())
                .padding(
                    start = VolnaTheme.tokens.spacing.md,
                    end = VolnaTheme.tokens.spacing.md,
                    bottom = VolnaTheme.tokens.spacing.md,
                ),
            horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.md),
        ) {
            Text(
                text = "Отменить запись?",
                modifier = Modifier.fillMaxWidth(),
                textAlign = TextAlign.Center,
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onSurface,
            )
            Text(
                text = messageText,
                modifier = Modifier.fillMaxWidth(),
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurface,
            )
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(
                        color = Color(0xFFF2F2F2),
                        shape = RoundedCornerShape(VolnaTheme.tokens.radius.lg),
                    )
                    .padding(VolnaTheme.tokens.spacing.sm),
                verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xs),
            ) {
                Text(
                    text = cancellationLabel,
                    modifier = Modifier
                        .background(
                            color = MaterialTheme.colorScheme.surface,
                            shape = RoundedCornerShape(VolnaTheme.tokens.radius.md),
                        )
                        .padding(horizontal = VolnaTheme.tokens.spacing.sm, vertical = VolnaTheme.tokens.spacing.xxs),
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = cancellationHint,
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurface,
                )
            }
            state.message?.let {
                Text(
                    text = it,
                    modifier = Modifier.fillMaxWidth(),
                    color = MaterialTheme.colorScheme.error,
                )
            }
            Button(
                onClick = { onIntent(BookingDetailsIntent.ConfirmCancel) },
                enabled = !state.isCancelling && kind != CancellationKind.UnavailableAfterStart,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp),
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = MaterialTheme.colorScheme.primary,
                    disabledContentColor = MaterialTheme.colorScheme.onSurfaceVariant,
                    containerColor = MaterialTheme.colorScheme.surface,
                    disabledContainerColor = MaterialTheme.colorScheme.surface,
                ),
                border = BorderStroke(1.dp, MaterialTheme.colorScheme.primary),
            ) {
                Text(if (state.isCancelling) "Отменяем..." else "Подтвердить отмену")
            }
            Button(
                onClick = { onIntent(BookingDetailsIntent.DismissCancel) },
                enabled = !state.isCancelling,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp),
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
                colors = ButtonDefaults.buttonColors(
                    containerColor = MaterialTheme.colorScheme.primary,
                    contentColor = MaterialTheme.colorScheme.onPrimary,
                ),
            ) {
                Text("Не отменять")
            }
            Spacer(Modifier.height(VolnaTheme.tokens.spacing.md))
        }
    }
}
