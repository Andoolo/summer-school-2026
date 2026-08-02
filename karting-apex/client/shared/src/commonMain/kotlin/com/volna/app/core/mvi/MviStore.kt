package com.volna.app.core.mvi

import kotlinx.coroutines.flow.StateFlow

interface MviStore<State, Intent, Effect> {
    val state: StateFlow<State>
    fun accept(intent: Intent)
    suspend fun effects(): Effect
}
