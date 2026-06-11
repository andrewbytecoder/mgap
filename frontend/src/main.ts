import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { createVuetify } from 'vuetify'
import 'vuetify/styles'
import '@mdi/font/css/materialdesignicons.css'
import './echarts'
import './style.css'

const pinia = createPinia()

const vuetify = createVuetify({
  theme: {
    defaultTheme: 'livePprof',
    themes: {
      livePprof: {
        dark: false,
        colors: {
          background: '#f5f6f0',
          surface: '#fffdf7',
          primary: '#304a3d',
          secondary: '#d86e3f',
          accent: '#3f6a8f',
          success: '#4d7c53',
          warning: '#c98928',
          error: '#b74d45'
        }
      }
    }
  },
  defaults: {
    VCard: {
      rounded: 'xl'
    },
    VTextField: {
      variant: 'outlined',
      density: 'comfortable'
    },
    VNumberInput: {
      variant: 'outlined',
      density: 'comfortable'
    }
  }
})

createApp(App).use(pinia).use(vuetify).mount('#app')
