import { createRouter, createWebHashHistory } from 'vue-router'
// import HomeView from '../views/HomeView.vue'


// 路由画面
import login from '@/views/layout/login.vue'
import monitor from '@/views/monitor/monitor.vue'

const router = createRouter({
    history: createWebHashHistory(),
    routes: [
        {
            path: '/',
            redirect: "/monitor",
        },
        {
            path: '/login',
            name: 'login',
            component: login,
        },
        {
            path: '/monitor',
            name: 'monitor',
            component: monitor,
        }

    ]
})

export default router
