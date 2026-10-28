import java.io.File
import java.util.Properties
import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

// Versão: a mesma do desktop (productVersion em cmd/admin/wails.json). O
// scripts/build-apk.ps1 passa-a com -PretalhoVersao (compila numa cópia
// espelhada, onde ../admin não existe); a correr no repositório lê-a direto.
val versao: String = (findProperty("retalhoVersao") as String?)
    ?: rootDir.resolve("../admin/wails.json").takeIf { it.exists() }?.readText()
        ?.let { Regex("\"productVersion\"\\s*:\\s*\"([^\"]+)\"").find(it)?.groupValues?.get(1) }
    ?: "0.0.0"
// 1.2.3 → 10203: sobe sempre que a versão sobe (o Android recusa instalar
// por cima uma versionCode mais baixa).
val codigoVersao: Int = versao.split('.', '-').take(3).map { it.toIntOrNull() ?: 0 }
    .let { (it + listOf(0, 0, 0)).take(3) }
    .let { (maj, min, pat) -> maj * 10000 + min * 100 + pat }

// Chave de assinatura das versões de distribuição: NUNCA no repositório.
// Procura, por ordem: variável RETALHO_ASSINATURA (caminho de um
// .properties), %LOCALAPPDATA%\gestao-retalho\chaves\assinatura.properties,
// ~/.gestao-retalho/assinatura.properties. O .properties tem storeFile,
// storePassword, keyAlias e keyPassword.
val assinatura: Properties? = listOfNotNull(
    System.getenv("RETALHO_ASSINATURA"),
    System.getenv("LOCALAPPDATA")?.let { "$it/gestao-retalho/chaves/assinatura.properties" },
    "${System.getProperty("user.home")}/.gestao-retalho/assinatura.properties",
).map(::File).firstOrNull { it.isFile }?.let { f -> Properties().apply { f.inputStream().use(::load) } }

android {
    namespace = "com.gestaoretalho.pos"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.gestaoretalho.pos"
        // Sunmi V2 Pro: Android 7.1.2 (API 25), só 32 bits.
        minSdk = 25
        targetSdk = 36
        versionCode = codigoVersao
        versionName = versao
        // armeabi-v7a: Sunmi V2 Pro (só 32 bits). arm64-v8a: telemóveis e
        // terminais modernos — os Xiaomi recentes já não correm 32 bits.
        ndk { abiFilters += listOf("armeabi-v7a", "arm64-v8a") }
    }

    signingConfigs {
        if (assinatura != null) {
            create("distribuicao") {
                storeFile = File(assinatura.getProperty("storeFile"))
                storePassword = assinatura.getProperty("storePassword")
                keyAlias = assinatura.getProperty("keyAlias")
                keyPassword = assinatura.getProperty("keyPassword")
            }
        }
    }

    buildTypes {
        release {
            // Sem minificação: a ponte JS (@JavascriptInterface) e o código
            // gerado pelo gomobile são chamados por nome.
            isMinifyEnabled = false
            signingConfig = signingConfigs.findByName("distribuicao")
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
}

kotlin {
    compilerOptions { jvmTarget.set(JvmTarget.JVM_17) }
}

// Uma versão de distribuição sem chave sairia sem assinatura (e o Android
// não a instala): falha logo, com a explicação.
tasks.matching { it.name == "assembleRelease" }.configureEach {
    doFirst {
        if (assinatura == null) {
            throw GradleException(
                "Sem chave de assinatura: criar assinatura.properties (ver cmd/android/README.md, \"APK de distribuição\")."
            )
        }
    }
}

dependencies {
    implementation("androidx.webkit:webkit:1.12.1")
    implementation("androidx.core:core:1.13.1")
    // Backend Go (gomobile bind), gerado por scripts/build-go.sh.
    implementation(files("libs/gobridge.aar"))
}
