import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import vueDevTools from 'vite-plugin-vue-devtools'
import VueSetupExtend from 'vite-plugin-vue-setup-extend'
import tailwindcss from '@tailwindcss/vite'



import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'

import legacy from '@vitejs/plugin-legacy'

// https://vite.dev/config/
export default defineConfig({
  optimizeDeps: {
    include: ['amfe-flexible']
  },
  plugins: [
    require('postcss-pxtorem')({
      rootValue: 37.5, // 设计稿 375px 基准
      propList: ['*'], // 所有属性转 rem
      selectorBlackList: [] // 无需忽略的选择器
    }),
    vue(),
    vueJsx(),
    vueDevTools(),
    VueSetupExtend(),
    // 配置 jQuery 插件的参数
    tailwindcss(),
    AutoImport({
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
    legacy({
      // 指定需要兼容的浏览器版本
      targets: ['Android >= 4.4', 'iOS >= 9'], // 根据您的用户群体调整
      // 或者使用查询字符串，与 package.json 中的 browserslist 配置一致
      // targets: ['> 1%', 'last 2 versions', 'not dead']
    })
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  server: {
    host: "0.0.0.0",
    port: 8702 // 端口
  },
  build: {
    target: 'es2015'
  }
})


