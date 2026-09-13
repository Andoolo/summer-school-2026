package com.volna.app.map

import androidx.compose.foundation.*
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.domain.model.MeetingPoint
import com.volna.app.domain.model.Route
import com.volna.app.domain.model.RouteType

// BS-004 / LOGIC-006: route map sheet shows a mock screenshot and hands off the meeting point to external maps.
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun RouteMapSheet(
    route: Route,
    meetingPoint: MeetingPoint,
    mapLauncher: MapLauncher = PlatformMapLauncher,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(
        skipPartiallyExpanded = true,
    )

    ModalBottomSheet(
        // Шторка живёт на уровне окна и не знает о колонке приложения: без ограничения
        // на десктопе она растягивалась шире экрана, из которого открыта.
        sheetMaxWidth = VolnaTheme.tokens.sizing.screenMaxWidth,
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        shape = RoundedCornerShape(
            topStart = VolnaTheme.tokens.radius.lg,
            topEnd = VolnaTheme.tokens.radius.lg,
        ),
        containerColor = MaterialTheme.colorScheme.surface,
        dragHandle = {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = VolnaTheme.tokens.spacing.xs),
                contentAlignment = androidx.compose.ui.Alignment.TopCenter,
            ) {
                Box(
                    modifier = Modifier
                        .fillMaxWidth(0.12f)
                        .height(4.dp)
                        .background(
                            color = VolnaTheme.tokens.colors.handle,
                            shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
                        ),
                )
            }
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
            verticalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.sm),
        ) {
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = androidx.compose.ui.Alignment.CenterVertically,
            ) {
                Text(
                    text = "Маршрут",
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                Text(
                    text = "Закрыть",
                    modifier = Modifier.clickable(role = Role.Button) { onDismiss() },
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.primary,
                )
            }
            Row(
                horizontalArrangement = Arrangement.spacedBy(VolnaTheme.tokens.spacing.xxs),
            ) {
                RouteMapTag(text = route.type.toUiText(), color = VolnaTheme.tokens.colors.tagRouteType, contentColor = VolnaTheme.tokens.colors.onTagRouteType)
                RouteMapTag(text = route.name, color = VolnaTheme.tokens.colors.tagRouteName, contentColor = VolnaTheme.tokens.colors.onTagRouteName)
            }
            RouteMapPreview(
                route = route,
                meetingPoint = meetingPoint,
                onOpenExternal = { mapLauncher.openExternalMap(meetingPoint) },
            )
            Text(
                text = "Заезд по трассе займёт ${route.durationMin} минут",
                modifier = Modifier.fillMaxWidth(),
                style = MaterialTheme.typography.labelMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
            )
            Button(
                onClick = { mapLauncher.buildRouteTo(meetingPoint) },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp),
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
                colors = ButtonDefaults.buttonColors(
                    containerColor = MaterialTheme.colorScheme.primary,
                    contentColor = MaterialTheme.colorScheme.onPrimary,
                ),
            ) {
                Text("Проложить маршрут")
            }
            Button(
                onClick = { mapLauncher.openExternalMap(meetingPoint) },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(56.dp),
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
                colors = ButtonDefaults.outlinedButtonColors(
                    contentColor = MaterialTheme.colorScheme.primary,
                    containerColor = MaterialTheme.colorScheme.surface,
                ),
                border = BorderStroke(1.dp, MaterialTheme.colorScheme.primary),
            ) {
                Text("Открыть в Яндекс.Картах")
            }
        }
    }
}

@Composable
private fun RouteMapTag(text: String, color: Color, contentColor: Color = MaterialTheme.colorScheme.onSurface) {
    Text(
        text = text,
        modifier = Modifier
            .background(color = color, shape = RoundedCornerShape(VolnaTheme.tokens.radius.sm))
            .padding(horizontal = VolnaTheme.tokens.spacing.xs, vertical = VolnaTheme.tokens.spacing.xxs),
        style = MaterialTheme.typography.labelMedium,
        color = contentColor,
    )
}

private fun RouteType.toUiText(): String = when (this) {
    RouteType.Novice -> "Новичковый"
    RouteType.Experienced -> "Опытный"
}
