import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      redirect: '/drive'
    },
    {
      path: '/drive',
      name: 'drive',
      component: () => import('@/views/DriveConfig.vue')
    },
    {
      path: '/point',
      name: 'point',
      component: () => import('@/views/PointConfig.vue')
    },
    {
      path: '/alarm',
      name: 'alarm',
      component: () => import('@/views/AlarmConfig.vue')
    },
    {
      path: '/history',
      name: 'history',
      component: () => import('@/views/HistoryConfig.vue')
    },
    {
      path: '/history-data',
      name: 'historyData',
      component: () => import('@/views/HistoryData.vue')
    }
  ]
})

export default router
