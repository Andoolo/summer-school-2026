package com.volna.app.catalog.presentation

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.core.ui.Loadable
import com.volna.app.domain.model.LeaderboardEntry
import com.volna.app.domain.model.RouteId
import com.volna.app.domain.model.TrackDirection
import com.volna.app.domain.model.TrackPassport
import com.volna.app.map.TrackMinimap

/**
 * Экран трассы (F5): схема ведёт, характеристики под ней.
 *
 * Длина круга, число поворотов, направление и главная прямая приходят с бэкенда
 * рассчитанными из той же геометрии, которая рисуется здесь на схеме, — поэтому
 * цифры и картинка не могут разойтись.
 */
@Composable
fun TrackScreen(
    routeId: RouteId,
    state: TrackState,
    onIntent: (TrackIntent) -> Unit,
    onBack: () -> Unit,
) {
    LaunchedEffect(routeId) {
        onIntent(TrackIntent.Load(routeId))
    }
    Box(Modifier.fillMaxSize()) {
        BackButton(onBack)
        ScreenTitle("Трасса")
        when (val passport = state.passport) {
            Loadable.Initial,
            Loadable.Loading -> {
                SkeletonCard(y = VolnaTheme.tokens.sizing.listCardTopY)
                SkeletonCard(y = VolnaTheme.tokens.sizing.listCardSecondY)
            }

            is Loadable.Content -> TrackContent(
                passport = passport.value,
                leaderboard = state.leaderboard,
            )

            is Loadable.Empty -> StateMessage(
                title = "Трасса недоступна",
                description = "Попробуйте вернуться к списку заездов",
                buttonText = "Назад",
                onClick = onBack,
            )

            is Loadable.Error -> StateMessage(
                title = "Не удалось загрузить",
                description = "Проверьте соединение и попробуйте снова",
                buttonText = "Повторить",
                artwork = StateArtwork.Error,
                onClick = { onIntent(TrackIntent.Retry) },
            )
        }
    }
}

@Composable
private fun TrackContent(
    passport: TrackPassport,
    leaderboard: List<LeaderboardEntry>,
) {
    val spacing = VolnaTheme.tokens.spacing
    LazyColumn(
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
            TrackMinimap(points = passport.geometry?.points.orEmpty(), height = 200.dp)
        }
        item { TrackHeader(passport) }
        item { TrackKeyMetrics(passport) }
        item { TrackSpecs(passport) }
        passport.record?.let { record ->
            item { TrackRecordCard(name = record.name, lapMs = record.lapMs) }
        }
        if (leaderboard.size > 1) {
            item { TrackLeaderboard(leaderboard) }
        }
    }
}

@Composable
private fun TrackHeader(passport: TrackPassport) {
    val spacing = VolnaTheme.tokens.spacing
    Column(verticalArrangement = Arrangement.spacedBy(spacing.xxs)) {
        TypePill(text = passport.type.toTagText())
        Text(
            text = passport.name,
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onSurface,
        )
        passport.subtitleText()?.let { subtitle ->
            Text(
                text = subtitle,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
            )
        }
    }
}

/**
 * Тип трассы отдельной «пилюлей», а не через SlotTag: там цвет текста жёстко
 * onSurface — тёмный, и на фирменном красном фоне он бы не читался.
 */
@Composable
private fun TypePill(text: String) {
    Text(
        text = text,
        modifier = Modifier
            .background(
                color = VolnaTheme.tokens.colors.brand,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
            )
            .padding(
                horizontal = VolnaTheme.tokens.spacing.sm,
                vertical = VolnaTheme.tokens.spacing.xxs,
            ),
        style = MaterialTheme.typography.labelMedium,
        fontWeight = FontWeight.Bold,
        color = VolnaTheme.tokens.colors.onBrand,
    )
}

/** Три главные цифры трассы — то, ради чего сюда заходят. */
@Composable
private fun TrackKeyMetrics(passport: TrackPassport) {
    val spacing = VolnaTheme.tokens.spacing
    Row(horizontalArrangement = Arrangement.spacedBy(spacing.xs)) {
        MetricTile(label = "Длина", value = "${passport.lengthM} м", modifier = Modifier.weight(1f))
        MetricTile(label = "Поворотов", value = "${passport.corners}", modifier = Modifier.weight(1f))
        MetricTile(label = "Прямая", value = "${passport.mainStraightM} м", modifier = Modifier.weight(1f))
    }
}

@Composable
private fun MetricTile(
    label: String,
    value: String,
    modifier: Modifier = Modifier,
) {
    val spacing = VolnaTheme.tokens.spacing
    Column(
        modifier = modifier
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.lg),
            )
            .padding(horizontal = spacing.sm, vertical = spacing.xs),
        verticalArrangement = Arrangement.spacedBy(spacing.xxs),
    ) {
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
        Text(
            text = value,
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onSurface,
        )
    }
}

/** Паспортные характеристики: только то, что реально заполнено. */
@Composable
private fun TrackSpecs(passport: TrackPassport) {
    val rows = buildList {
        add("Направление" to passport.direction.toText())
        passport.widthM?.let { add("Ширина полотна" to "${it.toDecimalText()} м") }
        passport.elevationM?.let { add("Перепад высот" to "${it.toDecimalText()} м") }
        passport.kartText()?.let { add("Карт" to it) }
        add("Длительность заезда" to "${passport.durationMin} мин")
        add("Картов на трассе" to "до ${passport.capacityCap}")
    }
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(VolnaTheme.tokens.spacing.xl),
            )
            .padding(VolnaTheme.tokens.spacing.md),
    ) {
        rows.forEachIndexed { index, (label, value) ->
            DetailsInfoRow(label = label, value = value, boldValue = true)
            if (index != rows.lastIndex) {
                Box(
                    Modifier
                        .padding(vertical = VolnaTheme.tokens.spacing.xs)
                        .fillMaxWidth()
                        .height(1.dp)
                        .background(VolnaTheme.tokens.colors.border),
                )
            }
        }
    }
}

/** Действующий рекорд круга — акцентный блок, единственный тёмный на экране. */
@Composable
private fun TrackRecordCard(name: String, lapMs: Int) {
    val spacing = VolnaTheme.tokens.spacing
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = MaterialTheme.colorScheme.onSurface,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.lg),
            )
            .padding(horizontal = spacing.md, vertical = spacing.sm),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(spacing.xxs)) {
            Text(
                text = "Рекорд круга",
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.surfaceVariant,
            )
            Text(
                text = lapMs.toLapTimeText(),
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Black,
                color = MaterialTheme.colorScheme.surface,
            )
        }
        Column(
            horizontalAlignment = Alignment.End,
            verticalArrangement = Arrangement.spacedBy(spacing.xxs),
        ) {
            Text(
                text = "Держит",
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.surfaceVariant,
            )
            Text(
                text = name,
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.surface,
            )
        }
    }
}

/** Остальные гонщики — рекордсмен уже показан выше, поэтому здесь со второго места. */
@Composable
private fun TrackLeaderboard(entries: List<LeaderboardEntry>) {
    val spacing = VolnaTheme.tokens.spacing
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(spacing.xl),
            )
            .padding(spacing.md),
        verticalArrangement = Arrangement.spacedBy(spacing.sm),
    ) {
        Text(
            text = "Остальные гонщики",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onSurface,
        )
        entries.drop(1).forEach { entry ->
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = "${entry.position}",
                    modifier = Modifier.width(24.dp),
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Black,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
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
    }
}

private fun TrackDirection.toText(): String = when (this) {
    TrackDirection.Clockwise -> "По часовой"
    TrackDirection.CounterClockwise -> "Против часовой"
}

/** «Асфальт · открыта в 2019» — пропускает незаполненные части паспорта. */
private fun TrackPassport.subtitleText(): String? {
    val parts = listOfNotNull(
        surface,
        openedYear?.let { "открыта в $it" },
    )
    return parts.takeIf { it.isNotEmpty() }?.joinToString(" · ")
}

private fun TrackPassport.kartText(): String? {
    val model = kartModel
    val power = kartPowerHp?.let { "$it л.с." }
    return listOfNotNull(model, power).takeIf { it.isNotEmpty() }?.joinToString(" · ")
}

/** 8.0 → «8», 4.2 → «4,2»: целые метры без хвоста, дробные — с запятой. */
private fun Double.toDecimalText(): String {
    val rounded = (this * 10).toLong()
    val whole = rounded / 10
    val tenth = rounded % 10
    return if (tenth == 0L) "$whole" else "$whole,$tenth"
}
