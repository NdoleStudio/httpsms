export default {
  extends: ['stylelint-config-standard'],
  overrides: [
    {
      files: ['**/*.vue'],
      extends: ['stylelint-config-recommended-vue'],
      customSyntax: 'postcss-html',
    },
  ],
  ignoreFiles: [
    '**/node_modules/**',
    '.nuxt/**',
    '.output/**',
    '.nitro/**',
    '.cache/**',
    'dist/**',
    'public/**',
  ],
  rules: {},
}
