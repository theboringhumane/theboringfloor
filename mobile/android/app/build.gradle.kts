import org.gradle.api.Action
import java.util.Properties

plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

val keystorePropertiesFile = rootProject.file("key.properties")
val expectedReleaseKeystorePath = "${System.getProperty("user.home")}/.theboringfloor/android/theboringfloor-release.jks"
val configuredReleaseKeystoreProperties = if (keystorePropertiesFile.exists()) {
    Properties().also { properties ->
        keystorePropertiesFile.inputStream().use(properties::load)
    }
} else {
    null
}

fun releaseSigningError(problem: String): Nothing = throw GradleException(
    "Release signing configuration error: $problem " +
        "Expected keystore: $expectedReleaseKeystorePath. " +
        "See ${keystorePropertiesFile.path}.example."
)

fun loadReleaseKeystoreProperties(): Properties {
    if (!keystorePropertiesFile.exists()) {
        releaseSigningError("missing ${keystorePropertiesFile.path}.")
    }

    return Properties().also { properties ->
        keystorePropertiesFile.inputStream().use(properties::load)
    }
}

android {
    namespace = "com.theboringhumane.theboringfloor"
    compileSdk = flutter.compileSdkVersion
    // Local NDK 28 is incomplete (missing source.properties); CI runners have different NDK sets,
    // so keep the working pin by default but allow -PtheboringfloorNdkVersion to override it.
    ndkVersion = providers.gradleProperty("theboringfloorNdkVersion")
        .orElse("27.0.12077973")
        .get()

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        // TODO: Specify your own unique Application ID (https://developer.android.com/studio/build/application-id.html).
        applicationId = "com.theboringhumane.theboringfloor"
        // You can update the following values to match your application needs.
        // For more information, see: https://flutter.dev/to/review-gradle-config.
        minSdk = flutter.minSdkVersion
        targetSdk = flutter.targetSdkVersion
        // Uses the version code from pubspec.yaml. When using split APKs, 1000 * ABI_VERSION
        // is added automatically by Flutter. (https://developer.android.com/studio/build/configure-apk-splits#configure-APK-versions)
        // You can force using the value of versionCode by specifying the `-P force-version-code-ignoring-abi=true`
        // flag during build.
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    signingConfigs {
        create("release") {
            // Placeholder values keep debug builds independent of release credentials;
            // release task-graph validation below rejects them before signing begins.
            keyAlias = configuredReleaseKeystoreProperties?.getProperty("keyAlias") ?: ""
            keyPassword = configuredReleaseKeystoreProperties?.getProperty("keyPassword") ?: ""
            storeFile = file(configuredReleaseKeystoreProperties?.getProperty("storeFile") ?: expectedReleaseKeystorePath)
            storePassword = configuredReleaseKeystoreProperties?.getProperty("storePassword") ?: ""
        }
    }

    buildTypes {
        release {
            signingConfig = signingConfigs.getByName("release")
        }
    }
}

gradle.taskGraph.whenReady(Action {
    if (allTasks.none { task -> task.name.contains("release", ignoreCase = true) }) {
        return@Action
    }

    val keystoreProperties = loadReleaseKeystoreProperties()
    val requiredProperties = listOf("storeFile", "storePassword", "keyAlias", "keyPassword")
    val missingProperty = requiredProperties.firstOrNull { property ->
        keystoreProperties.getProperty(property).isNullOrBlank()
    }
    if (missingProperty != null) {
        releaseSigningError("property '$missingProperty' is missing or blank in ${keystorePropertiesFile.path}.")
    }

    val configuredStoreFile = file(keystoreProperties.getProperty("storeFile"))
    if (!configuredStoreFile.exists()) {
        releaseSigningError("keystore file configured by 'storeFile' does not exist: ${configuredStoreFile.path}.")
    }

})

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

flutter {
    source = "../.."
}
