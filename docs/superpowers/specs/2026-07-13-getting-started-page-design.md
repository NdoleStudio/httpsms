# Getting Started page — Design

**Date:** 2026-07-13
**Status:** Approved

## Goal

Create the shortest possible, dummy-proof onboarding page that walks a
non-technical user through sending their first SMS with httpSMS:
**download → install → sign in → grant permissions → send a test message.**

The page must be self-explanatory and short. It reuses images from existing
blog posts and adds placeholders where the user will later drop GIFs/images.

## Route

- File: `web/app/pages/getting-started.vue`
- URL: `/getting-started`
- Layout: `website`

## Page structure (top → bottom)

1. **Hero**
   - `text-display-medium` (desktop) / `text-display-small` (mobile) title,
     e.g. "Send your first SMS in 5 minutes".
   - One-line subtitle.
   - Primary CTA `VBtn` (color primary): "⬇️ Download the Android app" linking
     to `https://github.com/NdoleStudio/httpsms/releases/latest/download/HttpSms.apk`.
   - Secondary CTA linking to `/login` (create account) when useful.
   - No gradient/glowing text (reserved for homepage hero).

2. **Numbered steps** (4 steps; each has a heading, 1–2 sentences, and an
   image or placeholder):
   - **Step 1 — Create your account & copy your API key.**
     Reuse `/img/blog/forward-incoming-sms-from-phone-to-webhook/settings.png`.
     Link to `/settings`.
   - **Step 2 — Download & install the app.**
     APK download link + placeholder for a GIF of installing the APK
     (`/img/getting-started/install-app.gif`). Info `VAlert` reminding to use
     international format e.g. `+18005550199`.
   - **Step 3 — Sign in on your phone.**
     Reuse `/img/blog/forward-incoming-sms-from-phone-to-webhook/android-app.png`;
     placeholder for a sign-in GIF (`/img/getting-started/sign-in.gif`).
   - **Step 4 — Send your first SMS.**
     From the web dashboard; link to `/threads`. Placeholder for a compose GIF
     (`/img/getting-started/send-sms.gif`).
   - After Step 3: an **info `VAlert`** linking to
     `/blog/grant-send-and-read-sms-permissions-on-android` for granting SMS
     permissions on Android 15+.

3. **"Automate it" section (bottom)**
   - Heading + links/cards to existing how-to blog posts:
     - Python — `/blog/send-sms-from-android-phone-with-python`
     - Excel — `/blog/how-to-send-sms-messages-from-excel`
     - CSV bulk — `/blog/send-bulk-sms-from-csv-file-with-no-code`
     - Zapier / Google Sheets — `/blog/send-sms-when-new-row-is-added-to-google-sheets-using-zapier`
     - Webhook forwarding — `/blog/forward-incoming-sms-from-phone-to-webhook`
     - End-to-end encryption — `/blog/end-to-end-encryption-to-sms-messages`

## Placeholders

For not-yet-available GIFs, render a styled bordered box (dashed border,
centered caption, e.g. "📹 GIF: Installing the APK — coming soon") that
references a real path under `/public/img/getting-started/`. This lets the
user drop assets in later without markup changes.

## Discoverability

- **Footer** (`web/app/layouts/website.vue`): add a "Getting Started" link to
  the **Resources** list.
- **`/threads` desktop empty state** (`web/app/pages/threads/index.vue`): add a
  "New here? Get started" `VBtn` below the Discord message (inside the
  `lgAndUp` empty-state block).

## Conventions to follow

- Vuetify 4 typography classes (`text-display-*`, `text-headline-large`,
  `text-title-large`, `text-body-large`).
- Every hyperlink: `text-decoration-none hover:text-decoration-underline`.
- `useSeoMeta` for title/description/OG tags (client-only SPA, `ssr: false`).
- Reuse `BlogSidebar`, `BackButton` patterns where appropriate.
- Warm, plain-language, beginner-friendly copy.

## Out of scope

- No "first login only" detection logic on `/threads`; the button shows in the
  existing desktop empty state.
- No new backend/API changes.
