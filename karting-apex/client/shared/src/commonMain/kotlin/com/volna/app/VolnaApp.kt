package com.volna.app

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import com.volna.app.auth.SessionRepository
import com.volna.app.auth.presentation.AuthEffect
import com.volna.app.auth.presentation.AuthIntent
import com.volna.app.auth.presentation.AuthStore
import com.volna.app.booking.presentation.*
import com.volna.app.catalog.presentation.*
import com.volna.app.core.config.AppConfig
import com.volna.app.core.navigation.BindBrowserNavigation
import com.volna.app.core.navigation.BindSystemBack
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.core.time.AppClock
import com.volna.app.profile.presentation.ProfileEffect
import com.volna.app.profile.presentation.ProfileIntent
import com.volna.app.profile.presentation.ProfileMode
import com.volna.app.profile.presentation.ProfileStore
import kotlinx.coroutines.launch
import androidx.navigation.NavDestination
import androidx.navigation.NavDestination.Companion.hasRoute
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import org.koin.compose.koinInject
import org.koin.compose.viewmodel.koinViewModel

@Composable
fun VolnaApp() {
    VolnaTheme {
        val appScope = rememberCoroutineScope()
        val navController = rememberNavController()
        val appConfig = koinInject<AppConfig>()
        val clock = koinInject<AppClock>()
        val sessionRepository = koinInject<SessionRepository>()
        val authStore = koinViewModel<AuthStore>()
        val profileStore = koinViewModel<ProfileStore>()
        val slotListStore = koinViewModel<SlotListStore>()
        val slotDetailsStore = koinViewModel<SlotDetailsStore>()
        val bookingFormStore = koinViewModel<BookingFormStore>()
        val bookingListStore = koinViewModel<BookingListStore>()
        val bookingDetailsStore = koinViewModel<BookingDetailsStore>()
        val trackStore = koinViewModel<TrackStore>()
        val authState by authStore.state.collectAsState()
        val profileState by profileStore.state.collectAsState()
        val slotListState by slotListStore.state.collectAsState()
        val slotDetailsState by slotDetailsStore.state.collectAsState()
        val bookingFormState by bookingFormStore.state.collectAsState()
        val bookingListState by bookingListStore.state.collectAsState()
        val bookingDetailsState by bookingDetailsStore.state.collectAsState()
        val trackState by trackStore.state.collectAsState()
        var rootState by remember { mutableStateOf(RootState.CheckingSession) }
        val backStackEntry by navController.currentBackStackEntryAsState()
        val currentDestination = backStackEntry?.destination

        fun resetToAuth() {
            appScope.launch {
                sessionRepository.clearToken()
            }
            authStore.accept(AuthIntent.Reset)
            profileStore.accept(ProfileIntent.Reset)
            slotListStore.accept(SlotListIntent.Reset)
            slotDetailsStore.accept(SlotDetailsIntent.Reset)
            bookingFormStore.accept(BookingFormIntent.Reset)
            bookingListStore.accept(BookingListIntent.Reset)
            bookingDetailsStore.accept(BookingDetailsIntent.Reset)
            trackStore.accept(TrackIntent.Reset)
            rootState = RootState.Ready
            navController.navigate(AuthDestination) {
                popUpTo(navController.graph.findStartDestination().id) {
                    inclusive = true
                }
                launchSingleTop = true
            }
        }

        fun canHandleSystemBack(): Boolean = when (rootState) {
            RootState.CheckingSession -> false

            RootState.Ready -> when {
                currentDestination?.hasRoute<AuthDestination>() == true ->
                    authState.step != com.volna.app.auth.presentation.AuthStep.Phone
                slotListState.filtersVisible -> true
                slotDetailsState.showRouteMap -> true
                bookingFormState.createdBooking != null -> true
                bookingDetailsState.showCancelConfirm -> true
                bookingDetailsState.showRouteMap -> true
                profileState.logoutConfirmVisible -> true
                profileState.deleteConfirmVisible -> true
                currentDestination?.hasRoute<SlotBookingDestination>() == true -> true
                currentDestination?.hasRoute<SlotDetailsDestination>() == true -> true
                currentDestination?.hasRoute<TrackDestination>() == true -> true
                currentDestination?.hasRoute<BookingDetailsDestination>() == true -> true
                currentDestination?.hasRoute<ProfileDestination>() == true &&
                    profileState.mode == ProfileMode.ConfirmPhone -> true
                currentDestination?.hasRoute<ProfileDestination>() == true &&
                    profileState.mode == ProfileMode.Edit -> true
                currentDestination?.hasRoute<SlotsDestination>() != true -> true
                else -> false
            }
        }

        fun handleSystemBack() = when (rootState) {
            RootState.CheckingSession -> false

            RootState.Ready -> when {
                currentDestination?.hasRoute<AuthDestination>() == true -> when (authState.step) {
                    com.volna.app.auth.presentation.AuthStep.Phone -> false
                    com.volna.app.auth.presentation.AuthStep.Otp,
                    com.volna.app.auth.presentation.AuthStep.Name,
                        -> {
                        authStore.accept(AuthIntent.BackToPhone)
                        true
                    }
                }

                slotListState.filtersVisible -> {
                    slotListStore.accept(SlotListIntent.CloseFilters)
                    true
                }

                slotDetailsState.showRouteMap -> {
                    slotDetailsStore.accept(SlotDetailsIntent.DismissRouteMap)
                    true
                }

                bookingFormState.createdBooking != null -> {
                    bookingFormStore.accept(BookingFormIntent.SuccessDismissed)
                    true
                }

                bookingDetailsState.showCancelConfirm -> {
                    bookingDetailsStore.accept(BookingDetailsIntent.DismissCancel)
                    true
                }

                bookingDetailsState.showRouteMap -> {
                    bookingDetailsStore.accept(BookingDetailsIntent.DismissRouteMap)
                    true
                }

                profileState.logoutConfirmVisible -> {
                    profileStore.accept(ProfileIntent.LogoutDismissed)
                    true
                }

                profileState.deleteConfirmVisible -> {
                    profileStore.accept(ProfileIntent.DeleteDismissed)
                    true
                }

                currentDestination?.hasRoute<SlotBookingDestination>() == true -> {
                    navController.popBackStack()
                    true
                }

                currentDestination?.hasRoute<SlotDetailsDestination>() == true -> {
                    navController.popBackStack()
                    true
                }

                currentDestination?.hasRoute<TrackDestination>() == true -> {
                    navController.popBackStack()
                    true
                }

                currentDestination?.hasRoute<BookingDetailsDestination>() == true -> {
                    navController.popBackStack()
                    true
                }

                currentDestination?.hasRoute<ProfileDestination>() == true &&
                    profileState.mode == ProfileMode.ConfirmPhone -> {
                    profileStore.accept(ProfileIntent.BackToEdit)
                    true
                }

                currentDestination?.hasRoute<ProfileDestination>() == true &&
                    profileState.mode == ProfileMode.Edit -> {
                    profileStore.accept(ProfileIntent.EditCancelled)
                    true
                }

                currentDestination?.hasRoute<SlotsDestination>() != true -> {
                    navController.navigate(SlotsDestination) {
                        popUpTo(navController.graph.findStartDestination().id) {
                            saveState = true
                        }
                        launchSingleTop = true
                        restoreState = true
                    }
                    true
                }

                else -> false
            }
        }

        BindBrowserNavigation(navController)
        BindSystemBack(enabled = canHandleSystemBack()) {
            handleSystemBack()
        }

        LaunchedEffect(sessionRepository) {
            if (sessionRepository.token().isNullOrBlank()) {
                rootState = RootState.Ready
                navController.navigate(AuthDestination) {
                    popUpTo(navController.graph.findStartDestination().id) {
                        inclusive = true
                    }
                    launchSingleTop = true
                }
            } else {
                rootState = RootState.Ready
                if (navController.currentDestination.isAuthOrMissing()) {
                    navController.navigate(SlotsDestination) {
                        popUpTo(navController.graph.findStartDestination().id) {
                            inclusive = true
                        }
                        launchSingleTop = true
                    }
                }
            }
        }

        LaunchedEffect(authStore) {
            while (true) {
                when (authStore.effects()) {
                    AuthEffect.Authenticated -> {
                        rootState = RootState.Ready
                        navController.navigate(SlotsDestination) {
                            popUpTo(navController.graph.findStartDestination().id) {
                                inclusive = true
                            }
                            launchSingleTop = true
                        }
                    }
                }
            }
        }

        LaunchedEffect(profileStore) {
            while (true) {
                when (profileStore.effects()) {
                    ProfileEffect.SignedOut -> resetToAuth()
                }
            }
        }

        LaunchedEffect(slotListStore) {
            while (true) {
                when (slotListStore.effects()) {
                    SlotListEffect.SignedOut -> resetToAuth()
                }
            }
        }

        LaunchedEffect(slotDetailsStore) {
            while (true) {
                when (slotDetailsStore.effects()) {
                    SlotDetailsEffect.SignedOut -> resetToAuth()
                }
            }
        }

        LaunchedEffect(trackStore) {
            while (true) {
                when (trackStore.effects()) {
                    TrackEffect.SignedOut -> resetToAuth()
                }
            }
        }

        LaunchedEffect(bookingFormStore) {
            while (true) {
                when (bookingFormStore.effects()) {
                    BookingFormEffect.SignedOut -> resetToAuth()
                }
            }
        }

        LaunchedEffect(bookingListStore) {
            while (true) {
                when (bookingListStore.effects()) {
                    BookingListEffect.SignedOut -> resetToAuth()
                }
            }
        }

        LaunchedEffect(bookingDetailsStore) {
            while (true) {
                when (bookingDetailsStore.effects()) {
                    BookingDetailsEffect.SignedOut -> resetToAuth()
                    BookingDetailsEffect.BookingChanged -> bookingListStore.accept(BookingListIntent.Refresh)
                }
            }
        }

        Box(Modifier.fillMaxSize()) {
            MainTabs(
                navController = navController,
                authState = authState,
                onAuthIntent = authStore::accept,
                slotListState = slotListState,
                onSlotListIntent = slotListStore::accept,
                slotDetailsState = slotDetailsState,
                onSlotDetailsIntent = slotDetailsStore::accept,
                trackState = trackState,
                onTrackIntent = trackStore::accept,
                bookingFormState = bookingFormState,
                onBookingFormIntent = bookingFormStore::accept,
                bookingListState = bookingListState,
                onBookingListIntent = bookingListStore::accept,
                bookingDetailsState = bookingDetailsState,
                onBookingDetailsIntent = bookingDetailsStore::accept,
                clock = clock,
                appConfig = appConfig,
                profileState = profileState,
                onProfileIntent = profileStore::accept,
            )
            if (rootState == RootState.CheckingSession) {
                SessionSplash()
            }
        }
    }
}

private fun NavDestination?.isAuthOrMissing(): Boolean =
    this == null || hasRoute<AuthDestination>()

@Composable
private fun SessionSplash() {
    Surface(
        modifier = Modifier.fillMaxSize(),
        color = MaterialTheme.colorScheme.background,
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(VolnaTheme.tokens.spacing.md),
            verticalArrangement = Arrangement.Center,
        ) {
            Text(
                text = "Апекс",
                style = MaterialTheme.typography.headlineLarge,
                fontWeight = FontWeight.Bold,
            )
        }
    }
}
