package com.volna.app.core.navigation

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.navigation.ExperimentalBrowserHistoryApi
import androidx.navigation.NavHostController
import androidx.navigation.bindToBrowserNavigation

@Composable
@OptIn(ExperimentalBrowserHistoryApi::class)
actual fun BindBrowserNavigation(navController: NavHostController) {
    LaunchedEffect(navController) {
        navController.bindToBrowserNavigation()
    }
}

@Composable
actual fun BindSystemBack(
    enabled: Boolean,
    onBack: () -> Unit,
) {
}
