# Affiliates Landing Page — Design

**Date:** 2026-07-05
**Component:** `web/` (Nuxt 4 SPA, Vuetify 4)

## Goal

Create a marketing landing page at `/affiliates` that convinces people to join the
httpSMS affiliate program, and replace the current external "Affiliates" footer link
(`https://httpsms.lemonsqueezy.com/affiliates`) with a `NuxtLink` to this new internal page.

All primary calls-to-action link to the LemonSqueezy affiliate signup:
`https://affiliates.lemonsqueezy.com/programs/httpsms` (opens in a new tab).

## Files

- **New:** `web/app/pages/affiliates/index.vue` — the landing page.
- **Edit:** `web/app/layouts/website.vue` — change the footer "Affiliates" link from the
  external `<a href="https://httpsms.lemonsqueezy.com/affiliates">` to
  `<NuxtLink to="/affiliates">`, keeping the existing `mdiShieldStar` warning icon and
  `text-white text-decoration-none footer-link` classes.

## Conventions to follow

- `<script setup lang="ts">` with `definePageMeta({ layout: 'website' })` and `useSeoMeta(...)`
  (mirrors `web/app/pages/privacy-policy/index.vue`).
- Vuetify 4 components (`VContainer`, `VRow`, `VCol`, `VCard`, `VBtn`, `VIcon`). No expansion
  panels. Use latest component props verified via context7 during implementation.
- Vuetify 4 typography classes only: `text-display-medium`, `text-headline-medium`,
  `text-headline-small`, `text-title-large`, `text-body-large` (no legacy `text-h*`).
- Hyperlinks get `text-decoration-none hover:text-decoration-underline` classes.
- Icons from `@mdi/js` imported in the script.
- External CTA links use `target="_blank"` and `rel="noopener"`.

## Page structure

The page uses the `website` layout (which provides the site nav + footer). Content is wrapped
in a `VContainer`; each section centered with `VCol cols="12" md="8" offset-md="2"` (or a wider
grid for the benefit cards).

Final copy below is written per the copywriting skill: benefit-first, specific, active voice,
customer language, one idea per section.

### 1. Hero
- Eyebrow text: `AFFILIATE PROGRAM`.
- H1 (`text-display-medium`): **Get paid to share httpSMS**
- Sub (`text-body-large`): *Earn up to $70 for every customer you refer — and keep earning every
  month they stay subscribed. If you already tell people to ditch expensive short codes, you might
  as well get paid for it.*
- Buttons: **Become an affiliate** (primary, → signup link) and **See how it works**
  (secondary/text, scrolls to the "How it works" section).

### 2. Why promote httpSMS (benefit cards)
Four `VCard`s in a responsive `VRow` (`cols=12 md=6` or `md=3`), each with an `@mdi/js` icon,
title, and one-line body:
1. **Up to $70 per sale** — Real commission on real subscriptions, not pennies per click.
2. **Recurring commission** — Get paid every month your referral stays, not just once.
3. **A product that sticks** — Developers wire httpSMS into daily workflows, so they rarely leave.
4. **Payouts on autopilot** — LemonSqueezy tracks every referral and pays you automatically.

### 3. How it works (3 numbered steps)
Section has an `id` so the hero "See how it works" button can scroll to it.
1. **Sign up free** — Join through LemonSqueezy in under a minute. No cost, no catch.
2. **Share your link** — Drop it in your blog posts, videos, docs, or DMs.
3. **Get paid** — We handle tracking and payouts. You collect on every sale.

### 4. FAQ (static two-column grid — no expansion panels)
Section heading (`text-headline-medium`): **Frequently asked questions**. Questions render
directly on the page (no accordions). Use a `VRow` of `VCol cols="12" md="6"` so it shows **two
columns on desktop and a single column on mobile**. Each item: bold question
(`text-title-large`) followed by the answer paragraph (`text-body-large`).

- **How much can I earn?** — Up to $70 per sale, plus recurring commission for as long as your
  referral stays subscribed.
- **How do I get paid?** — LemonSqueezy tracks your referrals and pays you automatically on their
  schedule.
- **Who can join?** — Anyone. Bloggers, YouTubers, developers, agencies — and it's free to join.
- **How do I track my referrals?** — Your LemonSqueezy dashboard shows clicks, referrals, and
  earnings in real time.
- **What converts best?** — Tutorials, honest reviews, and comparisons that show how httpSMS turns
  an Android phone into an SMS gateway.

### 5. Closing CTA banner
Full-width highlighted band:
- Heading: **Ready to start earning?**
- Sub: *Join free, grab your link, and turn your audience into recurring income.*
- Button: **Become an affiliate** → signup link.

## SEO meta

- `title`: "Affiliate Program — Earn up to $70 per Sale | httpSMS"
- `description`: "Join the httpSMS affiliate program and earn up to $70 for every customer you
  refer, with recurring commissions. Free to join, powered by LemonSqueezy."
- `ogTitle` / `ogDescription` mirror the above; `ogImage: https://httpsms.com/header.png`;
  `twitterCard: summary_large_image`.

## Out of scope

- No backend/API changes.
- No changes to the existing LemonSqueezy affiliate script in `nuxt.config.ts` /
  `public/integrations.js`.
- No new dependencies.

## Testing / verification

- `cd web && pnpm lint` passes for the changed files.
- Manual: `/affiliates` renders in the `website` layout; every CTA points to
  `https://affiliates.lemonsqueezy.com/programs/httpsms`; footer "Affiliates" link navigates to
  `/affiliates` internally.
