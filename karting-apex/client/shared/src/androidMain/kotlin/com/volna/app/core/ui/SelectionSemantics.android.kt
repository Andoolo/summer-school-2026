package com.volna.app.core.ui

import androidx.compose.ui.Modifier

// Состояние объявляет TalkBack по роли и selectable/toggleable — ничего не дописываем.
actual fun selectionLabel(label: String, selected: Boolean): String = label

actual fun Modifier.webSelectionLabel(label: String, selected: Boolean): Modifier = this

actual fun Modifier.webSwitchLabel(label: String, checked: Boolean): Modifier = this

actual fun Modifier.webLabel(label: String): Modifier = this
