package com.volna.app.core.navigation

import androidx.compose.runtime.Composable
import androidx.navigation.NavHostController

@Composable
expect fun BindBrowserNavigation(navController: NavHostController)

@Composable
expect fun BindSystemBack(
    enabled: Boolean,
    onBack: () -> Unit,
)
