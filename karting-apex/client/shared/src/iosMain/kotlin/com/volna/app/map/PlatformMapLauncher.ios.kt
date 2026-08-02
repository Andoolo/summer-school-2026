package com.volna.app.map

import com.volna.app.domain.model.MeetingPoint
import platform.Foundation.NSURL
import platform.UIKit.UIApplication

actual object PlatformMapLauncher : MapLauncher {
    actual override fun openExternalMap(meetingPoint: MeetingPoint) {
        open(meetingPoint.toExternalPointUrl())
    }

    actual override fun buildRouteTo(meetingPoint: MeetingPoint) {
        open(meetingPoint.toExternalRouteUrl())
    }

    private fun open(url: String) {
        NSURL.URLWithString(url)?.let { UIApplication.sharedApplication.openURL(it) }
    }
}
