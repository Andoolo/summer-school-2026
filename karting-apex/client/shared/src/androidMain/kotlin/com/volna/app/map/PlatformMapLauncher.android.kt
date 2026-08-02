package com.volna.app.map

import android.content.Context
import android.content.Intent
import android.net.Uri
import com.volna.app.core.logging.AppLogger
import com.volna.app.domain.model.MeetingPoint

actual object PlatformMapLauncher : MapLauncher {
    private var context: Context? = null

    fun initialize(context: Context) {
        this.context = context.applicationContext
    }

    actual override fun openExternalMap(meetingPoint: MeetingPoint) {
        val lat = meetingPoint.coordinates.lat
        val lng = meetingPoint.coordinates.lng
        val label = Uri.encode(meetingPoint.title.ifBlank { "SUP meeting point" })
        val pointUri = Uri.parse("geo:$lat,$lng?q=$lat,$lng($label)")
        open(pointUri)
    }

    actual override fun buildRouteTo(meetingPoint: MeetingPoint) {
        val lat = meetingPoint.coordinates.lat
        val lng = meetingPoint.coordinates.lng
        val routeUri = Uri.parse("google.navigation:q=$lat,$lng")
        open(routeUri)
    }

    private fun open(uri: Uri) {
        val appContext = context ?: return
        val intent = Intent(Intent.ACTION_VIEW, uri).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        runCatching { appContext.startActivity(intent) }
            .onFailure { failure -> AppLogger.e(failure, "Failed to open map URI: $uri") }
    }
}
