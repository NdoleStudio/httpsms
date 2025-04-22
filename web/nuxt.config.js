export default {
  // Target: https://go.nuxtjs.dev/config-target
  target: 'static',

  // Global page headers: https://go.nuxtjs.dev/config-head
  head: {
    titleTemplate: '%s',
    title: 'Convert your android phone into an SMS gateway - httpSMS',
    htmlAttrs: {
      lang: 'en',
    },
    script: [
      {
        hid: 'integrations',
        src: '/integrations.js',
        async: true,
        defer: true,
      },
      {
        hid: 'lemonsqueezy',
        src: 'https://lmsqueezy.com/affiliate.js',
        async: true,
        defer: true,
      },
      {
        hid: 'cloudflare',
        src: 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit',
      },
    ],
    meta: [
      { charset: 'utf-8' },
      { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      {
        hid: 'description',
        name: 'description',
        content:
          'Use your android phone to send and receive SMS messages using a simple HTTP API.',
      },
      { name: 'format-detection', content: 'telephone=no' },
      { hid: 'twitter:site', name: 'twitter:site', content: '@NdoleStudio' },
      {
        hid: 'twitter:card',
        name: 'twitter:card',
        content: 'summary_large_image',
      },
      {
        hid: 'og:title',
        name: 'og:title',
        content: 'Convert your android phone into an SMS gateway - httpSMS',
      },
      {
        hid: 'og:description',
        name: 'og:description',
        content:
          'Use your android phone to send and receive SMS messages using a simple HTTP API.',
      },
      {
        hid: 'og:image',
        name: 'og:image',
        content: 'https://httpsms.com/header.png',
      },
    ],
    link: [{ rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }],
  },

  // Global CSS: https://go.nuxtjs.dev/config-css
  css: [],

  // Plugins to run before rendering page: https://go.nuxtjs.dev/config-plugins
  plugins: [
    '~/plugins/filters.ts',
    { src: '~/plugins/vue-glow', ssr: false },
    { src: '~/plugins/chart', ssr: false },
  ],

  // Auto import components: https://go.nuxtjs.dev/config-components
  components: true,

  // Modules for dev and build (recommended): https://go.nuxtjs.dev/config-modules
  buildModules: [
    // https://go.nuxtjs.dev/typescript
    '@nuxt/typescript-build',
    // https://go.nuxtjs.dev/stylelint
    '@nuxtjs/stylelint-module',
    // https://go.nuxtjs.dev/vuetify
    '@nuxtjs/vuetify',
  ],

  // Modules: https://go.nuxtjs.dev/config-modules
  modules: [
    // Simple usage
    '@nuxtjs/dotenv',
    [
      '@nuxtjs/firebase',
      {
        config: {
          apiKey: process.env.FIREBASE_API_KEY,
          authDomain: process.env.FIREBASE_AUTH_DOMAIN,
          projectId: process.env.FIREBASE_PROJECT_ID,
          storageBucket: process.env.FIREBASE_STORAGE_BUCKET,
          messagingSenderId: process.env.FIREBASE_MESSAGING_SENDER_ID,
          appId: process.env.FIREBASE_APP_ID,
          measurementId: process.env.FIREBASE_MEASUREMENT_ID,
        },
        services: {
          analytics: true,
          auth: {
            persistence: 'local', // default
            initialize: {
              onAuthStateChangedAction: 'onAuthStateChanged',
              onIdTokenChangedAction: 'onIdTokenChanged',
              subscribeManually: false,
            },
            ssr: false,
          },
        },
      },
    ],
    // Simple Usage
    [
      'nuxt-highlightjs',
      {
        style: 'androidstudio',
      },
    ],
    '@nuxtjs/sitemap', // always put it at the end
  ],

  // Vuetify module configuration: https://go.nuxtjs.dev/config-vuetify
  vuetify: {
    treeShake: true,
    customVariables: ['~/assets/variables.scss'],
    defaultAssets: {
      icons: 'mdiSvg',
    },
    theme: {
      dark: true,
    },
  },

  sitemap: {
    hostname: 'https://httpsms.com',
    gzip: true,
    trailingSlash: true,
    exclude: [
      '/messages',
      '/settings',
      '/threads**',
      '/billing',
      '/bulk-messages',
    ],
  },

  publicRuntimeConfig: {
    checkoutURL: process.env.CHECKOUT_URL,
    enterpriseCheckoutURL: process.env.ENTERPRISE_CHECKOUT_URL,
    cloudflareTurnstileSiteKey: process.env.CLOUDFLARE_TURNSTILE_SITE_KEY,
    pusherKey: process.env.PUSHER_KEY,
    pusherCluster: process.env.PUSHER_CLUSTER,
  },

  // Build Configuration: https://go.nuxtjs.dev/config-build
  build: {
    transpile: ['chart.js', 'vue-chartjs'],
  },

  server: {
    port: 3000,
  },
}
