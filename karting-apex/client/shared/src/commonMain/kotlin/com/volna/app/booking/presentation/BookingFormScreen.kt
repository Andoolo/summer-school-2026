package com.volna.app.booking.presentation

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.domain.model.Booking
import com.volna.app.domain.model.Slot
import com.volna.app.domain.policy.BookingPriceCalculator
import com.volna.app.uikit.icons.Back
import com.volna.app.uikit.icons.Icons
import com.volna.app.uikit.icons.Info
import com.volna.app.uikit.icons.VolnaIcon

@Composable
fun BookingFormScreen(
    slot: Slot,
    state: BookingFormState,
    onIntent: (BookingFormIntent) -> Unit,
    onBack: () -> Unit,
    onDone: () -> Unit,
    onOpenBookings: () -> Unit,
) {
    LaunchedEffect(slot.id) {
        onIntent(BookingFormIntent.Open(slot))
    }
    Column(modifier = Modifier.fillMaxSize()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = VolnaTheme.tokens.spacing.md, vertical = 18.dp),
            verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.md),
        ) {
            CircleActionButton(icon = Icons.Back, contentDescription = "Назад", onClick = onBack)
            Text(
                text = "Оформление записи",
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onSurface,
            )
        }
        Box(modifier = Modifier.fillMaxSize()) {
            BookingFormContent(
                state = state,
                onIntent = onIntent,
            )
            state.createdBooking?.let { booking ->
                BookingSuccessSheet(
                    booking = booking,
                    fallbackPrice = state.totalPrice?.value ?: 0,
                    onDone = onDone,
                    onOpenBookings = onOpenBookings,
                )
            }
        }
    }
}

@Composable
private fun BookingFormContent(
    state: BookingFormState,
    onIntent: (BookingFormIntent) -> Unit,
) {
    val slot = state.slot
    val maxSeats = state.availability?.maxSeatsForBooking ?: 1
    val seatPrice = slot?.price?.value ?: 0
    val rentalPrice = slot?.rentalPrice?.value ?: 0
    val seatsTotal = seatPrice * state.seatsCount
    val rentalTotal = rentalPrice * state.rentalCount
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(horizontal = VolnaTheme.tokens.spacing.md),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.lg),
    ) {
        slot?.let {
            BookingSlotSummaryCard(it)
        }
        BookingSeatsCard(
            value = state.seatsCount,
            maxSeats = maxSeats,
            onMinus = { onIntent(BookingFormIntent.DecrementSeats) },
            onPlus = { onIntent(BookingFormIntent.IncrementSeats) },
        )
        if (slot != null) {
            BookingBoardsSection(
                boardSelections = state.boardSelections,
                freeRentalBoards = slot.freeRentalBoards,
                onBoardSelectionChange = { seatIndex, selection ->
                    onIntent(BookingFormIntent.SetBoardSelection(seatIndex, selection))
                },
            )
        }
        BookingPriceDetails(
            seatsCount = state.seatsCount,
            rentalCount = state.rentalCount,
            seatPrice = seatPrice,
            rentalPrice = rentalPrice,
            seatsTotal = seatsTotal,
            rentalTotal = rentalTotal,
            total = state.totalPrice?.value ?: 0,
        )
        state.validationMessage?.let {
            Text(it, color = MaterialTheme.colorScheme.error)
        }
        state.message?.let {
            Text(it, color = MaterialTheme.colorScheme.error)
        }
        Button(
            onClick = { onIntent(BookingFormIntent.Submit) },
            enabled = state.canSubmit,
            modifier = Modifier
                .fillMaxWidth()
                .height(VolnaTheme.tokens.sizing.buttonHeight),
            shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary),
        ) {
            Text(if (state.isSubmitting) "Записываем..." else "Записаться", fontWeight = FontWeight.Bold)
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
private fun BookingSlotSummaryCard(slot: Slot) {
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
            text = slot.startAt.toBookingCardStartText(),
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onSurface,
        )
        Row(horizontalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xxs)) {
            SlotTag(text = slot.route.type.toTagText(), color = Color(0xFF92FF9A))
            SlotTag(
                text = slot.route.name,
                color = Color(0xFFFFF897),
                modifier = Modifier.weight(1f, fill = false),
            )
            SlotTag(
                text = "Маршал: ${slot.instructor.name}",
                color = MaterialTheme.colorScheme.surface,
                modifier = Modifier.weight(1f, fill = false),
            )
        }
    }
}

@Composable
private fun BookingSeatsCard(
    value: Int,
    maxSeats: Int,
    onMinus: () -> Unit,
    onPlus: () -> Unit,
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
            text = "Число мест",
            modifier = Modifier.fillMaxWidth(),
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurface,
        )
        Row(
            horizontalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xs),
            verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
        ) {
            BookingCounterButton("−", onMinus)
            Text(
                text = value.toString(),
                modifier = Modifier
                    .width(52.dp)
                    .height(54.dp)
                    .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(VolnaTheme.tokens.radius.lg))
                    .padding(top = 16.dp),
                textAlign = TextAlign.Center,
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurface,
            )
            BookingCounterButton("+", onPlus)
        }
        Text(
            text = "Можно записать до $maxSeats мест",
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun BookingCounterButton(
    text: String,
    onClick: () -> Unit,
) {
    Text(
        text = text,
        modifier = Modifier
            .size(54.dp)
            .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(VolnaTheme.tokens.radius.lg))
            .border(1.dp, MaterialTheme.colorScheme.primary, RoundedCornerShape(VolnaTheme.tokens.radius.lg))
            .clickable { onClick() }
            .padding(top = 12.dp),
        textAlign = TextAlign.Center,
        style = MaterialTheme.typography.titleLarge,
        color = MaterialTheme.colorScheme.onSurface,
    )
}

@Composable
private fun BookingBoardsSection(
    boardSelections: List<BoardSelection>,
    freeRentalBoards: Int,
    onBoardSelectionChange: (Int, BoardSelection) -> Unit,
) {
    Column(verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm)) {
        Text(
            text = "Доска для каждого места",
            style = MaterialTheme.typography.bodyLarge,
            color = MaterialTheme.colorScheme.onSurface,
        )
        boardSelections.forEachIndexed { index, selection ->
            val seatNumber = index + 1
            BookingBoardRow(
                label = if (seatNumber == 1) "Место 1 (вы)" else "Место $seatNumber (гость)",
                rentalSelected = selection == BoardSelection.Rental,
                rentalEnabled = selection == BoardSelection.Rental ||
                        boardSelections.count { it == BoardSelection.Rental } < freeRentalBoards,
                onOwn = { onBoardSelectionChange(index, BoardSelection.Own) },
                onRental = { onBoardSelectionChange(index, BoardSelection.Rental) },
            )
        }
        Text(
            text = "Прокатных выбрано: ${boardSelections.count { it == BoardSelection.Rental }} из $freeRentalBoards",
            modifier = Modifier.fillMaxWidth(),
            textAlign = TextAlign.Center,
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun BookingBoardRow(
    label: String,
    rentalSelected: Boolean,
    rentalEnabled: Boolean,
    onOwn: () -> Unit,
    onRental: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(20.dp),
            )
            .padding(VolnaTheme.tokens.spacing.sm),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface,
        )
        Row(
            modifier = Modifier
                .border(1.dp, MaterialTheme.colorScheme.primary, RoundedCornerShape(VolnaTheme.tokens.radius.lg))
                .background(MaterialTheme.colorScheme.surface, RoundedCornerShape(VolnaTheme.tokens.radius.lg)),
        ) {
            BoardSegment(
                text = "Своя",
                selected = !rentalSelected,
                onClick = onOwn,
            )
            BoardSegment(
                text = "Прокатная",
                selected = rentalSelected,
                enabled = rentalEnabled,
                onClick = onRental,
            )
        }
    }
}

@Composable
private fun BoardSegment(
    text: String,
    selected: Boolean,
    enabled: Boolean = true,
    onClick: () -> Unit,
) {
    Text(
        text = text,
        modifier = Modifier
            .width(100.dp)
            .height(44.dp)
            .background(
                color = if (selected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surface,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.lg),
            )
            .clickable(enabled = enabled) { onClick() }
            .padding(top = 12.dp),
        textAlign = TextAlign.Center,
        style = MaterialTheme.typography.bodyMedium,
        color = when {
            selected -> MaterialTheme.colorScheme.onPrimary
            enabled -> MaterialTheme.colorScheme.onSurface
            else -> MaterialTheme.colorScheme.onSurfaceVariant
        },
    )
}

@Composable
private fun BookingPriceDetails(
    seatsCount: Int,
    rentalCount: Int,
    seatPrice: Int,
    rentalPrice: Int,
    seatsTotal: Int,
    rentalTotal: Int,
    total: Int,
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .border(width = 1.dp, color = MaterialTheme.colorScheme.surfaceVariant, shape = RoundedCornerShape(0.dp))
            .padding(top = VolnaTheme.tokens.spacing.sm),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
    ) {
        BookingPriceRow("Места: $seatPrice ₽ × $seatsCount", "$seatsTotal ₽")
        BookingPriceRow("Прокат: $rentalPrice ₽ × $rentalCount", "$rentalTotal ₽")
        BookingPriceRow("Итого", "$total ₽", bold = true)
    }
}

@Composable
private fun BookingPriceRow(
    label: String,
    value: String,
    bold: Boolean = false,
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.bodyLarge,
            fontWeight = if (bold) FontWeight.Bold else FontWeight.Normal,
            color = if (bold) MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Text(
            text = value,
            style = MaterialTheme.typography.bodyLarge,
            fontWeight = if (bold) FontWeight.Bold else FontWeight.Normal,
            color = if (bold) MaterialTheme.colorScheme.onSurface else MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun BookingSuccessSheet(
    booking: Booking,
    fallbackPrice: Int,
    onDone: () -> Unit,
    onOpenBookings: () -> Unit,
) {
    // CMP-15 / BS-002: successful createBooking summary; no network requests on open.
    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .padding(horizontal = VolnaTheme.tokens.spacing.md),
        horizontalAlignment = androidx.compose.ui.Alignment.CenterHorizontally,
    ) {
        Text(
            text = "Вы записаны",
            modifier = Modifier
                .fillMaxWidth()
                .offset(y = VolnaTheme.tokens.sizing.topTitleY),
            textAlign = TextAlign.Center,
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onSurface,
        )
        BookingSuccessSummaryCard(
            booking = booking,
            fallbackPrice = fallbackPrice,
            modifier = Modifier.offset(y = 142.dp),
        )
        Spacer(Modifier.weight(1f))
        Button(
            onClick = onOpenBookings,
            modifier = Modifier
                .fillMaxWidth()
                .height(VolnaTheme.tokens.sizing.buttonHeight),
            shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary),
        ) {
            Text("Мои бронирования", fontWeight = FontWeight.Bold)
        }
        OutlinedButton(
            onClick = onDone,
            modifier = Modifier
                .fillMaxWidth()
                .height(VolnaTheme.tokens.sizing.buttonHeight),
            shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
        ) {
            Text("Готово", fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.primary)
        }
        Box(
            modifier = Modifier
                .padding(top = VolnaTheme.tokens.spacing.lg, bottom = VolnaTheme.tokens.spacing.xs)
                .width(138.dp)
                .height(4.dp)
                .background(Color(0xFFCCCCCC), RoundedCornerShape(VolnaTheme.tokens.radius.pill)),
        )
    }
}

@Composable
private fun BookingSuccessSummaryCard(
    booking: Booking,
    fallbackPrice: Int,
    modifier: Modifier = Modifier,
) {
    val slot = booking.slot
    val total = BookingPriceCalculator.calculate(booking)?.value ?: fallbackPrice
    Column(
        modifier = modifier
            .width(VolnaTheme.tokens.sizing.contentWidth)
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(VolnaTheme.tokens.spacing.xl),
            )
            .padding(VolnaTheme.tokens.spacing.md),
        verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.lg),
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm)) {
            Text(
                text = slot?.startAt?.toBookingCardStartText() ?: "Запись создана",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.onSurface,
            )
            slot?.let {
                Row(horizontalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xxs)) {
                    SlotTag(text = it.route.type.toTagText(), color = Color(0xFF92FF9A))
                    SlotTag(
                        text = it.route.name,
                        color = Color(0xFFFFF897),
                        modifier = Modifier.weight(1f, fill = false),
                    )
                    SlotTag(
                        text = "Маршал: ${it.instructor.name}",
                        color = MaterialTheme.colorScheme.surface,
                        modifier = Modifier.weight(1f, fill = false),
                    )
                }
            }
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(
                        color = MaterialTheme.colorScheme.surface,
                        shape = RoundedCornerShape(VolnaTheme.tokens.radius.sm),
                    )
                    .padding(VolnaTheme.tokens.spacing.sm),
                verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
            ) {
                DetailsInfoRow("Мест", booking.seatsCount.toString())
                DetailsInfoRow("Доска в прокат", booking.rentalCount.toString())
            }
        }
        Column(verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xs)) {
            DetailsInfoRow("Итого", "$total ₽", boldValue = true)
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
}

@Composable
private fun CounterRow(
    title: String,
    value: Int,
    onMinus: () -> Unit,
    onPlus: () -> Unit,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .height(VolnaTheme.tokens.sizing.buttonHeight)
            .background(
                color = MaterialTheme.colorScheme.surface,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.lg),
            )
            .padding(horizontal = VolnaTheme.tokens.spacing.md),
        verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        Text(title, fontWeight = FontWeight.Bold)
        Row(
            horizontalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
            verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
        ) {
            Text(
                text = "−",
                modifier = Modifier
                    .size(VolnaTheme.tokens.spacing.xl)
                    .clickable { onMinus() },
                textAlign = TextAlign.Center,
                style = MaterialTheme.typography.headlineSmall,
            )
            Text(value.toString(), textAlign = TextAlign.Center, fontWeight = FontWeight.Bold)
            Text(
                text = "+",
                modifier = Modifier
                    .size(VolnaTheme.tokens.spacing.xl)
                    .clickable { onPlus() },
                textAlign = TextAlign.Center,
                style = MaterialTheme.typography.headlineSmall,
            )
        }
    }
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
private fun SlotTag(
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
