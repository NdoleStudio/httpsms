/**
 * Type declarations for Nuxt auto-imports.
 * These are normally generated in .nuxt/ by `nuxt prepare` but are provided
 * here so that external static analysis tools (e.g. Codacy) can resolve types
 * without running the Nuxt build pipeline.
 */

import type { $Fetch } from 'ofetch'
import type { App } from 'vue'

declare global {
  // Nuxt composables
  function useRuntimeConfig(): {
    public: Record<string, string>
    [key: string]: unknown
  }

  // Nuxt fetch utility
  const $fetch: $Fetch

  // Nuxt plugin helper
  function defineNuxtPlugin(
    plugin: (nuxtApp: { vueApp: App }) => Record<string, unknown> | undefined,
  ): unknown

  // Nuxt route middleware helper
  function defineNuxtRouteMiddleware(
    middleware: (to: {
      path: string
      query?: Record<string, string>
    }) => unknown,
  ): unknown

  // Nuxt navigation
  function navigateTo(
    to: string | { path: string; query?: Record<string, string | undefined> },
  ): unknown
}

export {}
