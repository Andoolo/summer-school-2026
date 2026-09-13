package com.volna.app.core.ui

import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics

actual fun selectionLabel(label: String, selected: Boolean): String =
    "$label, ${selectedStateText(selected)}"

actual fun Modifier.webSelectionLabel(label: String, selected: Boolean): Modifier =
    semantics { contentDescription = selectionLabel(label, selected) }

actual fun Modifier.webSwitchLabel(label: String, checked: Boolean): Modifier =
    semantics { contentDescription = "$label, ${switchStateText(checked)}" }

actual fun Modifier.webLabel(label: String): Modifier =
    semantics { contentDescription = label }
