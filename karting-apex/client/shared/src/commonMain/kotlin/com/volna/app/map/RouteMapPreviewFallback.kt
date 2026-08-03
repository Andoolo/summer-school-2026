package com.volna.app.map

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.domain.model.MeetingPoint
import com.volna.app.domain.model.Route

/**
 * Крупная схема трассы в шторке «Маршрут» с пином точки сбора.
 *
 * Раньше здесь рисовалась декоративная «карта города»: тёмная заливка-«вода», зелёные
 * пятна-«парки» и белые диагональные «улицы», поверх которых шла линия трассы. Улицы и парки
 * были бутафорией из «Волны» — они ничего не значили и спорили с настоящим контуром.
 * Теперь это та же схема, что и мини-карта на карточке заезда, только крупнее: одна
 * отрисовка на оба места, поэтому они не могут разъехаться по стилю.
 */
@Composable
fun RouteMapPreviewFallback(
    route: Route,
    meetingPoint: MeetingPoint,
    onOpenExternal: () -> Unit,
) {
    val spacing = VolnaTheme.tokens.spacing
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .padding(spacing.md),
        verticalArrangement = Arrangement.spacedBy(spacing.sm),
    ) {
        Text(
            text = "Адрес: ${meetingPoint.title.ifBlank { "уточняется" }}",
            modifier = Modifier.fillMaxWidth(),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurface,
            textAlign = TextAlign.Center,
        )
        TrackMinimap(
            points = route.geometry?.points.orEmpty(),
            modifier = Modifier.clickable { onOpenExternal() },
            height = 320.dp,
            cornerRadius = VolnaTheme.tokens.radius.md,
            trackWidth = 18.dp,
            padding = 32.dp,
            meetingPoint = meetingPoint.coordinates,
        )
    }
}
