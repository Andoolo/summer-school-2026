import androidx.compose.ui.window.ComposeUIViewController
import com.volna.app.VolnaApp
import com.volna.app.di.initKoin

fun MainViewController() = ComposeUIViewController {
    initKoin()
    VolnaApp()
}
