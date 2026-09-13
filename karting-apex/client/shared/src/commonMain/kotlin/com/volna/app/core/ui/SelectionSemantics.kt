package com.volna.app.core.ui

import androidx.compose.ui.Modifier

/**
 * Состояние «выбрано / включено» для скринридеров в веб-сборке.
 *
 * На Android и iOS роль (Tab, RadioButton, Checkbox, Switch) вместе с selectable/toggleable
 * даёт системе и роль, и состояние — скринридер сам скажет «выбрано». Compose для веба
 * (1.9) пока переносит в DOM только роль button и подпись (aria-label), без aria-checked
 * и aria-selected: слепой пользователь слышал бы «Тёмная, кнопка», не зная, какая тема
 * активна.
 *
 * Поэтому в wasm состояние дописывается в подпись. На остальных платформах функции
 * ничего не делают: там это продублировало бы системное объявление.
 */
expect fun selectionLabel(label: String, selected: Boolean): String

/** Модификатор-версия для элементов с текстом: в wasm задаёт подпись, иначе no-op. */
expect fun Modifier.webSelectionLabel(label: String, selected: Boolean): Modifier

/** То же для переключателей: «включено / выключено». */
expect fun Modifier.webSwitchLabel(label: String, checked: Boolean): Modifier

internal fun selectedStateText(selected: Boolean): String = if (selected) "выбрано" else "не выбрано"

internal fun switchStateText(checked: Boolean): String = if (checked) "включено" else "выключено"

/**
 * Подпись поля ввода для веба. У OutlinedTextField подпись задаётся через label, и на
 * Android/iOS скринридер её читает; Compose для веба (1.9) в aria-label её не переносит,
 * и поле объявлялось безымянным «текстовым полем». В wasm дублируем подпись в
 * contentDescription, на остальных платформах — no-op, чтобы не прочитать её дважды.
 */
expect fun Modifier.webLabel(label: String): Modifier
