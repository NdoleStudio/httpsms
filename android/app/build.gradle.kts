plugins {
    id("com.android.application")
    id("com.google.gms.google-services")
    id("io.sentry.android.gradle") version "6.2.0"
}

val gitHash = providers.exec {
    commandLine("git", "rev-parse", "--short", "HEAD")
}.standardOutput.asText.map { it.trim() }

android {
    compileSdk = 36

    defaultConfig {
        applicationId = "com.httpsms"
        minSdk = 28
        targetSdk = 36
        versionCode = 1
        versionName = gitHash.getOrElse("unknown")
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    buildTypes {
        getByName("debug") {
            manifestPlaceholders["sentryEnvironment"] = "development"
        }
        getByName("release") {
            manifestPlaceholders["sentryEnvironment"] = "production"
            isMinifyEnabled = false
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"), "proguard-rules.pro")
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_1_8
        targetCompatibility = JavaVersion.VERSION_1_8
    }
    namespace = "com.httpsms"

    buildFeatures {
        buildConfig = true
    }
}

dependencies {
    implementation(platform("com.google.firebase:firebase-bom:34.11.0"))
    implementation("com.journeyapps:zxing-android-embedded:4.3.0")
    implementation("com.google.firebase:firebase-analytics")
    implementation("com.google.firebase:firebase-messaging")
    implementation("com.squareup.okhttp3:okhttp:5.3.2")
    implementation("com.jakewharton.timber:timber:5.0.1")
    implementation("androidx.preference:preference-ktx:1.2.1")
    implementation("androidx.work:work-runtime-ktx:2.11.1")
    implementation("androidx.core:core-ktx:1.18.0")
    implementation("androidx.cardview:cardview:1.0.0")
    implementation("com.beust:klaxon:5.6")
    implementation("androidx.appcompat:appcompat:1.7.1")
    implementation("org.apache.commons:commons-text:1.15.0")
    implementation("com.google.android.material:material:1.13.0")
    implementation("androidx.constraintlayout:constraintlayout:2.2.1")
    implementation("com.googlecode.libphonenumber:libphonenumber:9.0.26")
    implementation("com.klinkerapps:android-smsmms:5.2.6")
    testImplementation("junit:junit:4.13.2")
    androidTestImplementation("androidx.test.ext:junit:1.3.0")
    androidTestImplementation("androidx.test.espresso:espresso-core:3.7.0")
}
