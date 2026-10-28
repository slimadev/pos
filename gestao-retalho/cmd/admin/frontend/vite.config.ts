import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'
import legacy from '@vitejs/plugin-legacy'
import postcssPresetEnv from 'postcss-preset-env'
import gapChrome62 from '../../android/web/postcss-gap-chrome62.js'

// O terminal Android (Sunmi V2 Pro, Android 7.1) tem uma WebView Chrome 62
// que não se consegue atualizar. `vite build --mode android` converte JS e
// CSS para essa versão (ver PLANO_ANDROID.md §0.2); o build do desktop fica
// exatamente como estava.
const CHROME_ANDROID = 'chrome >= 62'

// https://vitejs.dev/config/
export default defineConfig(({mode}) => {
  if (mode !== 'android') {
    return {plugins: [react()]}
  }
  return {
    base: './',
    plugins: [
      react(),
      legacy({targets: [CHROME_ANDROID], renderModernChunks: false}),
    ],
    css: {
      postcss: {
        plugins: [
          gapChrome62(),
          postcssPresetEnv({browsers: CHROME_ANDROID}),
        ],
      },
    },
    build: {outDir: 'dist-android'},
  }
})
