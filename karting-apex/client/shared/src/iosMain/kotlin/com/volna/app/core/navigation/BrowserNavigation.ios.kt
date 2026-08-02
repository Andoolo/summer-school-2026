package com.volna.app.core.navigation

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
}
