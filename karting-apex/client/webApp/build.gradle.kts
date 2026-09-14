plugins {
    alias(libs.plugins.kotlin.multiplatform)
    alias(libs.plugins.compose.multiplatform)
    alias(libs.plugins.compose.compiler)
}

@OptIn(org.jetbrains.kotlin.gradle.ExperimentalWasmDsl::class)
kotlin {
    wasmJs {
        outputModuleName.set("volna-web")
        browser {
            commonWebpackConfig {
                outputFileName = "volna-web.js"
            }
        }
        binaries.executable()
    }

    sourceSets {
        wasmJsMain.dependencies {
            implementation(project(":shared"))
            implementation(compose.runtime)
            implementation(compose.foundation)
            implementation(compose.material3)
            implementation(compose.ui)
            implementation(libs.kotlinx.browser)
            implementation(libs.skiko.js.wasm.runtime)
        }
    }
}

// Подставляет в собранный index.html предзагрузку обоих .wasm. Имена файлов содержат хеш
// содержимого и меняются от сборки к сборке, поэтому вписать их в исходный index.html
// нельзя. crossorigin (anonymous) совпадает с тем, как их запрашивает volna-web.js
// (fetch same-origin) — иначе браузер не переиспользовал бы предзагрузку и скачал бы
// файлы дважды.
val injectWasmPreload by tasks.registering {
    val distDir = layout.buildDirectory.dir("dist/wasmJs/productionExecutable")
    doLast {
        val dir = distDir.get().asFile
        val index = dir.resolve("index.html")
        val marker = "<!-- wasm-preload -->"
        val html = index.readText()
        check(marker in html) { "index.html: нет маркера $marker — предзагрузка wasm не подставлена" }
        val wasmFiles = dir.listFiles { file -> file.extension == "wasm" }.orEmpty().map { it.name }.sorted()
        check(wasmFiles.size == 2) { "ожидались 2 файла .wasm (Skiko и приложение), найдено: $wasmFiles" }
        val links = wasmFiles.joinToString("\n    ") { name ->
            "<link rel=\"preload\" href=\"$name\" as=\"fetch\" type=\"application/wasm\" crossorigin>"
        }
        index.writeText(html.replace(marker, links))
        logger.lifecycle("index.html: предзагрузка wasm — $wasmFiles")
    }
}

tasks.named("wasmJsBrowserDistribution") {
    finalizedBy(injectWasmPreload)
}

