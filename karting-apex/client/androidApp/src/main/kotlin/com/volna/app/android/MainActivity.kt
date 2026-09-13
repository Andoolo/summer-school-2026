package com.volna.app.android

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import com.volna.app.VolnaApp
import com.volna.app.core.storage.PlatformMarshalStorage
import com.volna.app.core.storage.PlatformSessionStorage
import com.volna.app.core.storage.PlatformThemeStorage
import com.volna.app.di.initKoin
import com.volna.app.map.PlatformMapLauncher
import timber.log.Timber

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        if (Timber.treeCount == 0) {
            Timber.plant(Timber.DebugTree())
        }
        PlatformSessionStorage.initialize(applicationContext)
        // Без initialize токен маршала жил только в памяти и терялся при перезапуске.
        PlatformMarshalStorage.initialize(applicationContext)
        // До initKoin: ThemeController читает тему при создании.
        PlatformThemeStorage.initialize(applicationContext)
        PlatformMapLauncher.initialize(applicationContext)
        initKoin()
        setContent {
            VolnaApp()
        }
    }
}
