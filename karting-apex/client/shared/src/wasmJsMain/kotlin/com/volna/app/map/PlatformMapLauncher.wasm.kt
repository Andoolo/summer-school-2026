package com.volna.app.map

import com.volna.app.domain.model.MeetingPoint
import kotlinx.browser.window

actual object PlatformMapLauncher : MapLauncher {
    actual override fun openExternalMap(meetingPoint: MeetingPoint) {
        window.open(meetingPoint.toExternalPointUrl(), "_blank")
    }

    actual override fun buildRouteTo(meetingPoint: MeetingPoint) {
        window.open(meetingPoint.toExternalRouteUrl(), "_blank")
    }
}
