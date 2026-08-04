package com.volna.app.core.navigation

import androidx.compose.runtime.Composable
import androidx.navigation.NavHostController

/** Адресной строки и системной кнопки «назад» на JVM-цели нет — привязывать нечего. */
@Composable
actual fun BindBrowserNavigation(navController: NavHostController) = Unit

@Composable
actual fun BindSystemBack(
    enabled: Boolean,
    onBack: () -> Unit,
) = Unit
