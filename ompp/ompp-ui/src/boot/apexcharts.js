import { defineBoot } from '#q-app/wrappers'
import VueApexCharts from 'vue3-apexcharts'
import en from 'apexcharts/dist/locales/en.json'
import fr from 'apexcharts/dist/locales/fr.json'

export default defineBoot(({ app }) => {
  // The app.use(VueApexCharts) will make <apexchart> component available everywhere
  app.use(VueApexCharts)
  app.config.globalProperties.chartLocales = [en, fr]
})
