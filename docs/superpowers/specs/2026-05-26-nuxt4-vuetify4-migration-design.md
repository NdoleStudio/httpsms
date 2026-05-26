# httpSMS Frontend Migration: Nuxt 2 + Vuetify 2 → Nuxt 4 + Vuetify 4

## Summary

Migrate the `web/` frontend from Nuxt 2 (Vue 2, Vuetify 2, Vuex, class-based components) to Nuxt 4 (Vue 3, Vuetify 4, Pinia, `<script setup lang="ts">`). Use a fresh Nuxt 4 project approach with incremental porting.

## Target Versions

| Library | Current | Target |
|---------|---------|--------|
| Nuxt | 2.18.1 | 4.x (latest) |
| Vue | 2.7.16 | 3.x (latest) |
| Vuetify | 2.7.2 | 4.x (latest) |
| State management | Vuex 3 | Pinia |
| Firebase | @nuxtjs/firebase | nuxt-vuefire |
| Component style | vue-property-decorator classes | `<script setup lang="ts">` |
| TypeScript | Partial | Full TypeScript everywhere |

## Decisions

- **Approach**: Fresh Nuxt 4 project with incremental file porting (not in-place upgrade)
- **Component style**: `<script setup lang="ts">` with Composition API
- **State**: Pinia stores split by domain
- **Firebase**: nuxt-vuefire module with composables
- **Rendering**: Static site generation (SSG)
- **Vuetify migration tool**: Use Vuetify MCP (`vuetify-mcp-get_upgrade_guide`, `vuetify-mcp-get_v4_breaking_changes`, `vuetify-mcp-get_component_api_by_version`) for every component to track breaking changes
- **Breakpoints**: Restore Vuetify 2/3 breakpoints via config (960/1280/1920/2560) to minimize layout changes

## Architecture

### Directory Structure (Nuxt 4)

```
web/
├── app/                        ← All client-side code
│   ├── assets/
│   │   ├── img/
│   │   ├── styles/
│   │   │   └── settings.scss   ← Vuetify SASS overrides
│   │   └── variables.scss
│   ├── components/
│   │   ├── BackButton.vue
│   │   ├── BlogAuthorBio.vue
│   │   ├── BlogInfo.vue
│   │   ├── CopyButton.vue
│   │   ├── FirebaseAuth.vue
│   │   ├── FixedHeader.vue
│   │   ├── LoadingButton.vue
│   │   ├── LoadingDashboard.vue
│   │   ├── MessageThread.vue
│   │   ├── MessageThreadHeader.vue
│   │   ├── NuxtLogo.vue
│   │   └── Toast.vue
│   ├── composables/
│   │   ├── useApi.ts           ← replaces plugins/axios.ts
│   │   ├── useAuth.ts          ← Firebase auth helpers
│   │   └── useNotification.ts  ← Toast/snackbar helpers
│   ├── layouts/
│   │   ├── default.vue
│   │   ├── error.vue
│   │   └── website.vue
│   ├── middleware/
│   │   ├── auth.ts
│   │   └── guest.ts
│   ├── pages/                  ← same structure, [id] instead of _id
│   │   ├── index.vue
│   │   ├── login.vue
│   │   ├── billing/index.vue
│   │   ├── blog/...
│   │   ├── bulk-messages/index.vue
│   │   ├── heartbeats/[id].vue
│   │   ├── messages/index.vue
│   │   ├── phone-api-keys/index.vue
│   │   ├── privacy-policy/index.vue
│   │   ├── search-messages/index.vue
│   │   ├── settings/index.vue
│   │   ├── terms-and-conditions/index.vue
│   │   └── threads/
│   │       ├── index.vue
│   │       └── [id]/index.vue
│   ├── plugins/
│   │   ├── vuetify.ts
│   │   ├── chart.client.ts
│   │   └── vue-glow.client.ts
│   ├── stores/                 ← Pinia stores
│   │   ├── auth.ts
│   │   ├── messages.ts
│   │   ├── threads.ts
│   │   ├── phones.ts
│   │   ├── billing.ts
│   │   ├── notifications.ts
│   │   └── app.ts
│   ├── utils/
│   │   ├── errors.ts
│   │   ├── filters.ts
│   │   ├── capitalize.ts
│   │   └── bag.ts
│   └── app.vue
├── shared/
│   └── types/                  ← API models
│       ├── api.ts
│       ├── billing.ts
│       ├── heartbeat.ts
│       ├── message-thread.ts
│       ├── message.ts
│       └── user.ts
├── public/                     ← renamed from static/
│   ├── favicon.ico
│   ├── integrations.js
│   ├── header.png
│   └── templates/
├── nuxt.config.ts
├── package.json
└── tsconfig.json
```

### Key Migration Patterns

#### Components: Class-based → `<script setup>`

**Before (Vue 2):**
```vue
<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'

@Component
export default class FirebaseAuth extends Vue {
  @Prop({ required: false, type: String, default: '/' }) to!: string
  firebaseUIInitialized = false

  mounted(): void { /* ... */ }
  beforeDestroy(): void { /* ... */ }
}
</script>
```

**After (Vue 3):**
```vue
<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'

const props = withDefaults(defineProps<{
  to?: string
}>(), { to: '/' })

const firebaseUIInitialized = ref(false)

onMounted(() => { /* ... */ })
onBeforeUnmount(() => { /* ... */ })
</script>
```

#### Vuetify Breakpoints: `$vuetify.breakpoint` → `useDisplay()`

**Before:**
```vue
<v-col :class="{ 'text-center': $vuetify.breakpoint.mdAndDown }">
```

**After:**
```vue
<script setup lang="ts">
import { useDisplay } from 'vuetify'
const { mdAndDown, lgAndUp } = useDisplay()
</script>
<template>
  <v-col :class="{ 'text-center': mdAndDown }">
</template>
```

#### State: Vuex → Pinia

**Before:**
```ts
this.$store.dispatch('loadPhones', true)
this.$store.getters.getAuthUser
```

**After:**
```ts
const phonesStore = usePhonesStore()
await phonesStore.loadPhones(true)
phonesStore.authUser
```

#### Firebase: `this.$fire.auth` → VueFire composables

**Before:**
```ts
await this.$fire.auth.currentUser?.getIdToken()
```

**After:**
```ts
import { useCurrentUser } from 'vuefire'
const user = useCurrentUser()
const token = await user.value?.getIdToken()
```

#### Dynamic Routes: `_id` → `[id]`

- `pages/threads/_id/index.vue` → `pages/threads/[id]/index.vue`
- `pages/heartbeats/_id.vue` → `pages/heartbeats/[id].vue`

### Vuetify 4 Breaking Changes to Address

Using the Vuetify MCP for each component, the key changes are:

1. **CSS Layers** — mandatory in v4; adjust any custom style overrides
2. **Theme** — default is now "system" (we want dark, configure explicitly)
3. **Typography** — MD2 → MD3 type scale (text-h1 → text-display-large, etc.)
4. **Breakpoints** — reduced default sizes (restore v3 values via config)
5. **Elevation** — 25 levels → 6 levels (MD3)
6. **VBtn** — no default uppercase, grid → flex layout
7. **VSnackbar** — removed multi-line prop
8. **VSelect** — "item" slot → "internalItem"
9. **Grid** — v-row/v-col overhauled
10. **CSS Reset** — mostly removed, add selective resets

### Vuetify MCP Usage Per Component

For EVERY component/page being migrated, the implementation must:
1. Call `vuetify-mcp-get_component_api_by_version` for each Vuetify component used
2. Call `vuetify-mcp-get_v4_breaking_changes` filtered by relevant category
3. Apply the correct v4 API (props, slots, events) based on MCP output
4. Verify no deprecated props/events remain

### Pinia Store Design

Split the monolithic Vuex store into domain stores:

| Store | Responsibility |
|-------|---------------|
| `auth.ts` | Firebase auth state, user profile, onAuthStateChanged |
| `messages.ts` | Messages CRUD, search |
| `threads.ts` | Message threads, current thread |
| `phones.ts` | Phone list, heartbeats, polling |
| `billing.ts` | Usage, subscription, payments |
| `notifications.ts` | Toast/snackbar queue |
| `app.ts` | App metadata, polling state, runtime config |

### Plugin Migrations

| Old Plugin | New Approach |
|-----------|-------------|
| `plugins/axios.ts` | `composables/useApi.ts` using `$fetch` with auth header |
| `plugins/filters.ts` | `utils/filters.ts` (import explicitly or app.config globalProperties) |
| `plugins/vue-glow.ts` | `plugins/vue-glow.client.ts` (client-only plugin) |
| `plugins/chart.ts` | `plugins/chart.client.ts` (client-only plugin) |
| `plugins/errors.ts` | `utils/errors.ts` |
| `plugins/bag.ts` | `utils/bag.ts` |
| `plugins/capitalize.ts` | `utils/capitalize.ts` |
| `plugins/veutify.ts` | `plugins/vuetify.ts` (createVuetify setup) |

## Migration Order (Tasks)

### Phase 1: Scaffold & Configuration
1. Initialize fresh Nuxt 4 project in `web/` (backup old code)
2. Install dependencies (vuetify, pinia, nuxt-vuefire, sass, @mdi/js, pusher-js, etc.)
3. Configure `nuxt.config.ts` (SSG, runtime config, modules)
4. Set up Vuetify plugin with dark theme, restored breakpoints, MDI SVG icons
5. Set up nuxt-vuefire with Firebase config
6. Configure TypeScript strictly

### Phase 2: Foundation
7. Port `shared/types/` (API models — mostly copy)
8. Port `utils/` (errors, filters, bag, capitalize)
9. Create `composables/useApi.ts` (replace Axios plugin)
10. Create `composables/useAuth.ts` (Firebase auth helpers)

### Phase 3: State Management
11. Create Pinia store: `stores/auth.ts`
12. Create Pinia store: `stores/notifications.ts`
13. Create Pinia store: `stores/app.ts`
14. Create Pinia store: `stores/phones.ts`
15. Create Pinia store: `stores/messages.ts`
16. Create Pinia store: `stores/threads.ts`
17. Create Pinia store: `stores/billing.ts`

### Phase 4: Layouts & Middleware
18. Port `middleware/auth.ts`
19. Port `middleware/guest.ts`
20. Port `layouts/default.vue` (with Vuetify MCP)
21. Port `layouts/website.vue` (with Vuetify MCP)
22. Port `layouts/error.vue` (with Vuetify MCP)
23. Create `app.vue`

### Phase 5: Components (use Vuetify MCP for each)
24. Port `components/Toast.vue`
25. Port `components/LoadingDashboard.vue`
26. Port `components/LoadingButton.vue`
27. Port `components/BackButton.vue`
28. Port `components/CopyButton.vue`
29. Port `components/FixedHeader.vue`
30. Port `components/BlogAuthorBio.vue`
31. Port `components/BlogInfo.vue`
32. Port `components/NuxtLogo.vue`
33. Port `components/FirebaseAuth.vue`
34. Port `components/MessageThread.vue`
35. Port `components/MessageThreadHeader.vue`

### Phase 6: Pages (use Vuetify MCP for each)
36. Port `pages/index.vue` (homepage)
37. Port `pages/login.vue`
38. Port `pages/threads/index.vue`
39. Port `pages/threads/[id]/index.vue`
40. Port `pages/messages/index.vue`
41. Port `pages/search-messages/index.vue`
42. Port `pages/bulk-messages/index.vue`
43. Port `pages/settings/index.vue`
44. Port `pages/billing/index.vue`
45. Port `pages/heartbeats/[id].vue`
46. Port `pages/phone-api-keys/index.vue`
47. Port `pages/privacy-policy/index.vue`
48. Port `pages/terms-and-conditions/index.vue`
49. Port `pages/blog/index.vue`
50. Port `pages/blog/how-to-send-sms-messages-from-excel.vue`
51. Port `pages/blog/grant-send-and-read-sms-permissions-on-android.vue`
52. Port `pages/blog/forward-incoming-sms-from-phone-to-webhook.vue`
53. Port `pages/blog/end-to-end-encryption-to-sms-messages.vue`
54. Port `pages/blog/send-bulk-sms-from-csv-file-with-no-code.vue`
55. Port `pages/blog/send-sms-from-android-phone-with-python.vue`
56. Port `pages/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier.vue`

### Phase 7: Final Setup
57. Port static assets (`public/`)
58. Port environment files (`.env`, `.env.production`)
59. Update Dockerfile and nginx.conf
60. Update sitemap configuration
61. Configure highlight.js (nuxt-highlightjs or manual)

### Phase 8: Verification (EVERY component and page)
62. Verify `app.vue` renders
63. Verify `layouts/default.vue` renders correctly
64. Verify `layouts/website.vue` renders correctly
65. Verify `layouts/error.vue` renders correctly
66. Verify `components/Toast.vue` renders correctly
67. Verify `components/LoadingDashboard.vue` renders correctly
68. Verify `components/LoadingButton.vue` renders correctly
69. Verify `components/BackButton.vue` renders correctly
70. Verify `components/CopyButton.vue` renders correctly
71. Verify `components/FixedHeader.vue` renders correctly
72. Verify `components/BlogAuthorBio.vue` renders correctly
73. Verify `components/BlogInfo.vue` renders correctly
74. Verify `components/NuxtLogo.vue` renders correctly
75. Verify `components/FirebaseAuth.vue` renders correctly
76. Verify `components/MessageThread.vue` renders correctly
77. Verify `components/MessageThreadHeader.vue` renders correctly
78. Verify `pages/index.vue` renders correctly
79. Verify `pages/login.vue` renders correctly
80. Verify `pages/threads/index.vue` renders correctly
81. Verify `pages/threads/[id]/index.vue` renders correctly
82. Verify `pages/messages/index.vue` renders correctly
83. Verify `pages/search-messages/index.vue` renders correctly
84. Verify `pages/bulk-messages/index.vue` renders correctly
85. Verify `pages/settings/index.vue` renders correctly
86. Verify `pages/billing/index.vue` renders correctly
87. Verify `pages/heartbeats/[id].vue` renders correctly
88. Verify `pages/phone-api-keys/index.vue` renders correctly
89. Verify `pages/privacy-policy/index.vue` renders correctly
90. Verify `pages/terms-and-conditions/index.vue` renders correctly
91. Verify `pages/blog/index.vue` renders correctly
92. Verify all blog subpages render correctly
93. Run `pnpm build` (static generation) successfully
94. Verify no TypeScript errors (`pnpm typecheck`)
95. Verify lint passes (`pnpm lint`)

## Verification Strategy

Each verification task in Phase 8 means:
1. Start the dev server (`pnpm dev`)
2. Navigate to the page/route in question
3. Confirm no console errors, no hydration mismatches
4. Confirm visual layout matches intent (Vuetify components render, dark theme active, responsive breakpoints work)
5. For interactive components (forms, modals, auth), confirm basic interactions work

The build verification (`pnpm build`) confirms all pages can be statically generated without errors.

## Risk Mitigations

- **Backup old code**: Keep old `web/` contents in a branch before starting
- **Incremental porting**: Each file is ported and verified before moving to the next
- **Vuetify MCP**: Use for every Vuetify component to catch breaking changes
- **Restored breakpoints**: Keep v2/v3 breakpoint values to minimize layout drift
- **CSS Reset compatibility**: Add selective reset CSS to maintain existing spacing behavior
