import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'

// 通用字体
import 'vfonts/Lato.css'
// 等宽字体
import 'vfonts/FiraCode.css'
import naive from "naive-ui";
// import axios from 'axios'
// import { http_Front_url } from "@/typer/index"

import "bootstrap"
import 'bootstrap/dist/css/bootstrap.min.css'
import 'bootstrap/dist/js/bootstrap.min.js'
import 'bootstrap/dist/js/bootstrap.bundle.min.js'


// import Antd from 'ant-design-vue';
// import 'ant-design-vue/dist/reset.css';



import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import zhCn from 'element-plus/es/locale/lang/zh-cn'

import 'es-drager/lib/style.css'
import Drager from 'es-drager'


const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(naive)
app.use(ElementPlus)
app.use(ElementPlus, { locale: zhCn })


for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
    app.component(key, component)
}
app.component('es-drager', Drager)
app.mount('#app')



// const Cloud_configure_token: string = localStorage.getItem('Cloud_configure_token') || "null"
// var token_info = JSON.parse(Cloud_configure_token) as Cloud_configure_token_interface;
// axios.defaults.headers.common['Authorization'] = token_info.Expires_in
