package com.volna.app.di

import com.volna.app.auth.AuthRepository
import com.volna.app.auth.SessionRepository
import com.volna.app.auth.data.DefaultSessionRepository
import com.volna.app.auth.data.KtorAuthRepository
import com.volna.app.auth.presentation.AuthStore
import com.volna.app.booking.BookingRepository
import com.volna.app.booking.IdempotencyKeyFactory
import com.volna.app.booking.data.KtorBookingRepository
import com.volna.app.booking.data.RandomIdempotencyKeyFactory
import com.volna.app.booking.presentation.BookingDetailsStore
import com.volna.app.booking.presentation.BookingFormStore
import com.volna.app.booking.presentation.BookingListStore
import com.volna.app.catalog.InstructorRepository
import com.volna.app.catalog.SlotRepository
import com.volna.app.catalog.data.KtorInstructorRepository
import com.volna.app.catalog.data.KtorSlotRepository
import com.volna.app.catalog.presentation.SlotDetailsStore
import com.volna.app.catalog.presentation.SlotListStore
import com.volna.app.core.config.AppConfig
import com.volna.app.core.network.VolnaApiClient
import com.volna.app.core.storage.PlatformSessionStorage
import com.volna.app.core.storage.SessionStorage
import com.volna.app.core.time.AppClock
import com.volna.app.core.time.SystemAppClock
import com.volna.app.profile.ProfileRepository
import com.volna.app.profile.data.KtorProfileRepository
import com.volna.app.profile.presentation.ProfileStore
import org.koin.core.context.startKoin
import org.koin.core.module.dsl.viewModel
import org.koin.dsl.module
import org.koin.mp.KoinPlatformTools

fun initKoin() {
    if (KoinPlatformTools.defaultContext().getOrNull() != null) return
    startKoin {
        modules(volnaAppModule)
    }
}

val volnaAppModule = module {
    single { AppConfig() }
    single<AppClock> { SystemAppClock }
    single<SessionStorage> { PlatformSessionStorage }
    single<SessionRepository> { DefaultSessionRepository(get()) }
    single { VolnaApiClient(get()) }

    single<AuthRepository> { KtorAuthRepository(get(), get()) }
    single<ProfileRepository> { KtorProfileRepository(get(), get()) }
    single<SlotRepository> { KtorSlotRepository(get()) }
    single<InstructorRepository> { KtorInstructorRepository(get()) }
    single<BookingRepository> { KtorBookingRepository(get()) }
    single<IdempotencyKeyFactory> { RandomIdempotencyKeyFactory() }

    viewModel { AuthStore(get(), get()) }
    viewModel { ProfileStore(get(), get()) }
    viewModel { SlotListStore(get(), get()) }
    viewModel { SlotDetailsStore(get()) }
    viewModel { BookingFormStore(get(), get()) }
    viewModel { BookingListStore(get(), get()) }
    viewModel { BookingDetailsStore(get(), get()) }
}
