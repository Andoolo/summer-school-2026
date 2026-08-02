package com.volna.app.map

import androidx.compose.runtime.Composable
import com.volna.app.domain.model.MeetingPoint
import com.volna.app.domain.model.Route

@Composable
actual fun RouteMapPreview(
    route: Route,
    meetingPoint: MeetingPoint,
    onOpenExternal: () -> Unit,
) {
    RouteMapPreviewFallback(route, meetingPoint, onOpenExternal)
}
