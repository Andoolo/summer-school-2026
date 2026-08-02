package com.volna.app.map

import androidx.compose.runtime.Composable
import com.volna.app.domain.model.MeetingPoint
import com.volna.app.domain.model.Route

interface MapLauncher {
    fun openExternalMap(meetingPoint: MeetingPoint)
    fun buildRouteTo(meetingPoint: MeetingPoint)
}

expect object PlatformMapLauncher : MapLauncher {
    override fun openExternalMap(meetingPoint: MeetingPoint)
    override fun buildRouteTo(meetingPoint: MeetingPoint)
}

internal fun MeetingPoint.toExternalPointUrl(): String {
    val lat = coordinates.lat
    val lng = coordinates.lng
    return "https://maps.apple.com/?ll=$lat,$lng&q=${title.toMapQuery()}"
}

internal fun MeetingPoint.toExternalRouteUrl(): String {
    val lat = coordinates.lat
    val lng = coordinates.lng
    return "https://maps.apple.com/?daddr=$lat,$lng&q=${title.toMapQuery()}"
}

private fun String.toMapQuery(): String =
    trim().ifBlank { "SUP meeting point" }.replace(" ", "+")

@Composable
expect fun RouteMapPreview(
    route: Route,
    meetingPoint: MeetingPoint,
    onOpenExternal: () -> Unit,
)
