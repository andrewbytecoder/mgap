import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify'
import Components from 'unplugin-vue-components/vite'
import Fonts from 'unplugin-fonts/vite'

export default defineConfig({
  plugins: [
    vue(),
    vuetify({ autoImport: true }),
    Components(),
    Fonts({
      google: {
        families: [
          {
            name: 'Space Grotesk',
            styles: 'wght@400;500;700'
          },
          {
            name: 'IBM Plex Mono',
            styles: 'wght@400;500'
          }
        ]
      }
    })
  ],
  server: {
    port: 34115,
    strictPort: true
  }
})
