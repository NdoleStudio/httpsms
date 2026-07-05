import withNuxt from './.nuxt/eslint.config.mjs'
import eslintConfigPrettier from 'eslint-config-prettier'

export default withNuxt(eslintConfigPrettier).append({
  name: 'httpsms/link-checker-overrides',
  rules: {
    // Links to static download assets in /public (e.g. /templates/*.xlsx) are
    // valid at runtime but are not Nuxt routes, so the static ESLint checks
    // false-positive on them. The build-time link checker still validates links.
    'link-checker/valid-route': 'off',
    'link-checker/valid-sitemap-link': 'off',
  },
})
