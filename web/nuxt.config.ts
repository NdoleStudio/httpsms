// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: "2025-01-01",

  ssr: false,

  modules: ["vuetify-nuxt-module", "@pinia/nuxt", "@nuxtjs/google-fonts"],

  googleFonts: {
    families: {
      Roboto: [100, 300, 400, 500, 700, 900],
    },
    display: "swap",
    download: true,
  },

  css: ["vuetify/styles"],

  build: {
    transpile: ["vuetify", "chart.js", "vue-chartjs", "v-phone-input"],
  },

  vite: {
    define: {
      "process.env.DEBUG": false,
    },
    optimizeDeps: {
      include: [
        "@mdi/js",
        "chartjs-adapter-moment",
        "date-fns",
        "firebase/app",
        "firebase/auth",
        "highlight.js/lib/core",
        "libphonenumber-js",
        "pusher-js",
        "qrcode",
      ],
    },
  },

  vuetify: {
    vuetifyOptions: {
      theme: {
        defaultTheme: "dark",
      },
      icons: {
        defaultSet: "mdi-svg",
      },
    },
  },

  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.API_BASE_URL || "http://localhost:8000",
      appUrl: process.env.APP_URL || "https://httpsms.com",
      appName: process.env.APP_NAME || "HTTP SMS",
      appGithubUrl:
        process.env.APP_GITHUB_URL || "https://github.com/NdoleStudio/httpsms",
      appDocumentationUrl:
        process.env.APP_DOCUMENTATION_URL || "https://docs.httpsms.com",
      appDownloadUrl:
        process.env.APP_DOWNLOAD_URL || "https://apk.httpsms.com/HttpSms.apk",
      appEnv: process.env.APP_ENV || "production",
      checkoutUrl: process.env.CHECKOUT_URL || "",
      enterpriseCheckoutUrl: process.env.ENTERPRISE_CHECKOUT_URL || "",
      cloudflareTurnstileSiteKey:
        process.env.CLOUDFLARE_TURNSTILE_SITE_KEY || "",
      pusherKey: process.env.PUSHER_KEY || "",
      pusherCluster: process.env.PUSHER_CLUSTER || "",
      firebaseApiKey: process.env.FIREBASE_API_KEY || "",
      firebaseAuthDomain: process.env.FIREBASE_AUTH_DOMAIN || "",
      firebaseProjectId: process.env.FIREBASE_PROJECT_ID || "",
      firebaseStorageBucket: process.env.FIREBASE_STORAGE_BUCKET || "",
      firebaseMessagingSenderId: process.env.FIREBASE_MESSAGING_SENDER_ID || "",
      firebaseAppId: process.env.FIREBASE_APP_ID || "",
      firebaseMeasurementId: process.env.FIREBASE_MEASUREMENT_ID || "",
    },
  },

  nitro: {
    prerender: {
      routes: [],
      failOnError: false,
    },
  },

  routeRules: {},

  app: {
    head: {
      titleTemplate: "%s",
      title: "Convert your android phone into an SMS gateway - httpSMS",
      htmlAttrs: { lang: "en" },
      script: [
        { src: "/integrations.js", async: true, defer: true },
        {
          src: "https://lmsqueezy.com/affiliate.js",
          async: true,
          defer: true,
        },
        {
          src: "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit",
        },
      ],
      meta: [
        { charset: "utf-8" },
        { name: "viewport", content: "width=device-width, initial-scale=1" },
        {
          name: "description",
          content:
            "Use your android phone to send and receive SMS messages using a simple HTTP API.",
        },
        { name: "format-detection", content: "telephone=no" },
        { name: "twitter:site", content: "@NdoleStudio" },
        { name: "twitter:card", content: "summary_large_image" },
        {
          property: "og:title",
          content: "Convert your android phone into an SMS gateway - httpSMS",
        },
        {
          property: "og:description",
          content:
            "Use your android phone to send and receive SMS messages using a simple HTTP API.",
        },
        {
          property: "og:image",
          content: "https://httpsms.com/header.png",
        },
      ],
      link: [{ rel: "icon", type: "image/x-icon", href: "/favicon.ico" }],
    },
  },
});
