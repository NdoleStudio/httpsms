// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: "2025-01-01",

  ssr: false,

  modules: ["vuetify-nuxt-module", "nuxt-vuefire", "@pinia/nuxt"],

  css: ["vuetify/styles"],

  build: {
    transpile: ["vuetify", "chart.js", "vue-chartjs"],
  },

  vite: {
    define: {
      "process.env.DEBUG": false,
    },
    css: {
      preprocessorOptions: {
        scss: {
          api: "modern-compiler",
        },
      },
    },
    optimizeDeps: {
      include: [
        "chartjs-adapter-moment",
        "@mdi/js",
        "date-fns",
        "libphonenumber-js",
        "firebase/compat/app",
        "firebase/compat/auth",
        "firebaseui",
        "qrcode",
        "pusher-js",
      ],
    },
  },

  vuetify: {
    moduleOptions: {
      styles: { configFile: "assets/styles/settings.scss" },
    },
    vuetifyOptions: {
      theme: {
        defaultTheme: "dark",
      },
      icons: {
        defaultSet: "mdi-svg",
      },
      display: {
        thresholds: {
          md: 960,
          lg: 1280,
          xl: 1920,
          xxl: 2560,
        },
      },
    },
  },

  vuefire: {
    config: {
      apiKey: process.env.FIREBASE_API_KEY,
      authDomain: process.env.FIREBASE_AUTH_DOMAIN,
      projectId: process.env.FIREBASE_PROJECT_ID,
      storageBucket: process.env.FIREBASE_STORAGE_BUCKET,
      messagingSenderId: process.env.FIREBASE_MESSAGING_SENDER_ID,
      appId: process.env.FIREBASE_APP_ID,
      measurementId: process.env.FIREBASE_MEASUREMENT_ID,
    },
    auth: {
      enabled: true,
      sessionCookie: false,
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
