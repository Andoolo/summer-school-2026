package com.volna.app.map

import androidx.compose.runtime.Composable
import com.volna.app.domain.model.MeetingPoint
import com.volna.app.domain.model.Route

/** Внешних карт на JVM-цели нет: она нужна только для запуска общих тестов. */
actual object PlatformMapLauncher : MapLauncher {
    actual override fun openExternalMap(meetingPoint: MeetingPoint) = Unit

    actual override fun buildRouteTo(meetingPoint: MeetingPoint) = Unit
}

/** Схему трассы рисует общий код (TrackMinimap), платформенной части здесь нет. */
@Composable
actual fun RouteMapPreview(
    route: Route,
    meetingPoint: MeetingPoint,
    onOpenExternal: () -> Unit,
) = RouteMapPreviewFallback(route = route, meetingPoint = meetingPoint, onOpenExternal = onOpenExternal)
