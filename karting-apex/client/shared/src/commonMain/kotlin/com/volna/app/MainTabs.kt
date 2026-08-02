package com.volna.app

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.navigation.NavDestination
import androidx.navigation.NavDestination.Companion.hasRoute
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.toRoute
import com.volna.app.booking.presentation.BookingDetailsIntent
import com.volna.app.booking.presentation.BookingDetailsScreen
import com.volna.app.booking.presentation.BookingDetailsState
import com.volna.app.booking.presentation.BookingFormIntent
import com.volna.app.booking.presentation.BookingFormScreen
import com.volna.app.booking.presentation.BookingFormState
import com.volna.app.booking.presentation.BookingListIntent
import com.volna.app.booking.presentation.BookingListScreen
import com.volna.app.booking.presentation.BookingListState
import com.volna.app.catalog.presentation.SlotDetailsIntent
import com.volna.app.catalog.presentation.SlotDetailsScreen
import com.volna.app.catalog.presentation.SlotDetailsState
import com.volna.app.catalog.presentation.SlotListIntent
import com.volna.app.catalog.presentation.SlotListScreen
import com.volna.app.catalog.presentation.SlotListState
import com.volna.app.core.config.AppConfig
import com.volna.app.core.theme.VolnaTheme
import com.volna.app.core.time.AppClock
import com.volna.app.core.ui.Loadable
import com.volna.app.profile.presentation.ProfileIntent
import com.volna.app.profile.presentation.ProfileScreen
import com.volna.app.profile.presentation.ProfileState
import com.volna.app.uikit.icons.Calendar
import com.volna.app.uikit.icons.Icons
import com.volna.app.uikit.icons.Options
import com.volna.app.uikit.icons.Profile
import com.volna.app.uikit.icons.VolnaIcon
import com.volna.app.auth.presentation.AuthIntent
import com.volna.app.auth.presentation.AuthScreen
import com.volna.app.auth.presentation.AuthState

@Composable
internal fun MainTabs(
    navController: NavHostController,
    authState: AuthState,
    onAuthIntent: (AuthIntent) -> Unit,
    slotListState: SlotListState,
    onSlotListIntent: (SlotListIntent) -> Unit,
    slotDetailsState: SlotDetailsState,
    onSlotDetailsIntent: (SlotDetailsIntent) -> Unit,
    bookingFormState: BookingFormState,
    onBookingFormIntent: (BookingFormIntent) -> Unit,
    bookingListState: BookingListState,
    onBookingListIntent: (BookingListIntent) -> Unit,
    bookingDetailsState: BookingDetailsState,
    onBookingDetailsIntent: (BookingDetailsIntent) -> Unit,
    clock: AppClock,
    appConfig: AppConfig,
    profileState: ProfileState,
    onProfileIntent: (ProfileIntent) -> Unit,
) {
    val backStackEntry by navController.currentBackStackEntryAsState()
    val currentDestination = backStackEntry?.destination
    val selectedTab = currentDestination.mainTab()

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background),
        contentAlignment = Alignment.TopCenter,
    ) {
        Box(
            modifier = Modifier
                .fillMaxSize()
                .widthIn(max = VolnaTheme.tokens.sizing.screenMaxWidth),
        ) {
            NavHost(
                navController = navController,
                startDestination = AuthDestination,
            ) {
                composable<AuthDestination> {
                    AuthScreen(
                        state = authState,
                        onIntent = onAuthIntent,
                    )
                }

                composable<SlotsDestination> {
                    SlotListScreen(
                        state = slotListState,
                        onIntent = onSlotListIntent,
                        onSlotClick = { slot ->
                            navController.navigate(SlotDetailsDestination(slot.id.value))
                        },
                    )
                }

                composable<SlotDetailsDestination> { entry ->
                    val route = entry.toRoute<SlotDetailsDestination>()
                    SlotDetailsScreen(
                        slotId = route.slotId(),
                        state = slotDetailsState,
                        onIntent = onSlotDetailsIntent,
                        onBack = { navController.popBackStack() },
                        onBook = { slot -> navController.navigate(SlotBookingDestination(slot.id.value)) },
                    )
                }

                composable<SlotBookingDestination> { entry ->
                    val route = entry.toRoute<SlotBookingDestination>()
                    val loadedSlot = (slotDetailsState.slot as? Loadable.Content)?.value
                        ?.takeIf { it.id == route.slotId() }
                    if (loadedSlot == null) {
                        SlotDetailsScreen(
                            slotId = route.slotId(),
                            state = slotDetailsState,
                            onIntent = onSlotDetailsIntent,
                            onBack = { navController.popBackStack() },
                            onBook = { slot -> navController.navigate(SlotBookingDestination(slot.id.value)) },
                        )
                    } else {
                        BookingFormScreen(
                            slot = loadedSlot,
                            state = bookingFormState,
                            onIntent = onBookingFormIntent,
                            onBack = { navController.popBackStack() },
                            onDone = {
                                onBookingFormIntent(BookingFormIntent.SuccessDismissed)
                                onSlotListIntent(SlotListIntent.Retry)
                                navController.navigate(SlotsDestination) {
                                    popUpTo<SlotsDestination> { inclusive = false }
                                    launchSingleTop = true
                                }
                            },
                            onOpenBookings = {
                                onBookingFormIntent(BookingFormIntent.SuccessDismissed)
                                onSlotListIntent(SlotListIntent.Retry)
                                onBookingListIntent(BookingListIntent.Refresh)
                                navController.navigate(BookingsDestination) {
                                    popUpTo<SlotsDestination> { inclusive = false }
                                    launchSingleTop = true
                                }
                            },
                        )
                    }
                }

                composable<BookingsDestination> {
                    BookingListScreen(
                        state = bookingListState,
                        onIntent = onBookingListIntent,
                        onBookingClick = { bookingId ->
                            navController.navigate(BookingDetailsDestination(bookingId.value))
                        },
                        onBookWalk = {
                            navController.navigate(SlotsDestination) {
                                popUpTo(navController.graph.findStartDestination().id) {
                                    saveState = true
                                }
                                launchSingleTop = true
                                restoreState = true
                            }
                        },
                    )
                }

                composable<BookingDetailsDestination> { entry ->
                    val route = entry.toRoute<BookingDetailsDestination>()
                    BookingDetailsScreen(
                        bookingId = route.bookingId(),
                        state = bookingDetailsState,
                        clock = clock,
                        onIntent = onBookingDetailsIntent,
                        onBack = { navController.popBackStack() },
                    )
                }

                composable<ProfileDestination> {
                    ProfileScreen(
                        state = profileState,
                        appConfig = appConfig,
                        onIntent = onProfileIntent,
                    )
                }
            }

            if (isNavBarVisible(
                    currentDestination = currentDestination,
                    slotListState = slotListState,
                    slotDetailsState = slotDetailsState,
                    bookingFormState = bookingFormState,
                    bookingDetailsState = bookingDetailsState,
                    profileState = profileState,
                )
            ) {
                FloatingNavigationBar(
                    selectedTab = selectedTab,
                    onTabSelected = { tab ->
                        navController.navigate(tab.destination()) {
                            popUpTo(navController.graph.findStartDestination().id) {
                                saveState = true
                            }
                            launchSingleTop = true
                            restoreState = true
                        }
                    },
                    modifier = Modifier.align(Alignment.BottomCenter),
                )
            }
        }
    }
}

private fun isNavBarVisible(
    currentDestination: NavDestination?,
    slotListState: SlotListState,
    slotDetailsState: SlotDetailsState,
    bookingFormState: BookingFormState,
    bookingDetailsState: BookingDetailsState,
    profileState: ProfileState,
): Boolean = when {
    currentDestination?.hasRoute<AuthDestination>() == true -> false
    currentDestination?.hasRoute<SlotDetailsDestination>() == true -> false
    currentDestination?.hasRoute<SlotBookingDestination>() == true -> false
    currentDestination?.hasRoute<BookingDetailsDestination>() == true -> false
    slotListState.filtersVisible -> false
    slotDetailsState.showRouteMap -> false
    bookingFormState.createdBooking != null -> false
    bookingDetailsState.showCancelConfirm -> false
    bookingDetailsState.showRouteMap -> false
    profileState.logoutConfirmVisible -> false
    profileState.deleteConfirmVisible -> false
    else -> true
}

private fun NavDestination?.mainTab(): MainTab = when {
    this?.hasRoute<BookingsDestination>() == true -> MainTab.Bookings
    this?.hasRoute<BookingDetailsDestination>() == true -> MainTab.Bookings
    this?.hasRoute<ProfileDestination>() == true -> MainTab.Profile
    else -> MainTab.Slots
}

@Composable
private fun FloatingNavigationBar(
    selectedTab: MainTab,
    onTabSelected: (MainTab) -> Unit,
    modifier: Modifier = Modifier,
) {
    Row(
        modifier = modifier
            .padding(bottom = VolnaTheme.tokens.sizing.navBottomPadding)
            .width(VolnaTheme.tokens.sizing.navWidth)
            .height(VolnaTheme.tokens.sizing.navHeight)
            .shadow(
                elevation = VolnaTheme.tokens.spacing.sm,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
            )
            .background(
                color = MaterialTheme.colorScheme.surface,
                shape = RoundedCornerShape(VolnaTheme.tokens.radius.pill),
            )
            .padding(horizontal = VolnaTheme.tokens.spacing.lg),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
    ) {
        NavItem(
            tab = MainTab.Slots,
            selected = selectedTab == MainTab.Slots,
            icon = Icons.Calendar,
            onClick = onTabSelected,
        )
        NavItem(
            tab = MainTab.Bookings,
            selected = selectedTab == MainTab.Bookings,
            icon = Icons.Options,
            onClick = onTabSelected,
        )
        NavItem(
            tab = MainTab.Profile,
            selected = selectedTab == MainTab.Profile,
            icon = Icons.Profile,
            onClick = onTabSelected,
        )
    }
}

@Composable
private fun NavItem(
    tab: MainTab,
    selected: Boolean,
    icon: ImageVector,
    onClick: (MainTab) -> Unit,
) {
    VolnaIcon(
        imageVector = icon,
        contentDescription = tab.title,
        modifier = Modifier.clickable { onClick(tab) },
        tint = if (selected) {
            MaterialTheme.colorScheme.primary
        } else {
            MaterialTheme.colorScheme.onSurfaceVariant
        },
        size = VolnaTheme.tokens.spacing.xl,
    )
}
