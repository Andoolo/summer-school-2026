package com.volna.app.core.navigation

import androidx.activity.compose.BackHandler
import androidx.compose.runtime.Composable
import androidx.navigation.NavHostController

@Composable
actual fun BindBrowserNavigation(navController: NavHostController) {
}

@Composable
actual fun BindSystemBack(
    enabled: Boolean,
    onBack: () -> Unit,
) {
    BackHandler(enabled = enabled, onBack = onBack)
}
