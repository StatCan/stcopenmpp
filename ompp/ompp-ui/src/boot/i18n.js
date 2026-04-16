import { defineBoot } from '#q-app/wrappers'
import { createI18n } from 'vue-i18n'
import messages from 'src/i18n'

// import translations for Quasar controls
import * as fr from 'quasar/lang/fr.js'
import * as enUS from 'quasar/lang/en-US.js'

export default defineBoot(({ app }) => {
  const i18n = createI18n({
    legacy: true, // required to use the Options API / Legacy API methods
    locale: 'en-US',
    fallbackLocale: 'en-US',
    silentFallbackWarn: true,
    // silentTranslationWarn: process.env.PROD,
    silentTranslationWarn: true,
    globalInjection: true,
    messages
  })

  // Set i18n instance on app
  app.use(i18n)
  app.config.globalProperties.$quasarLangs = [enUS, fr] // Quasar translations
})
