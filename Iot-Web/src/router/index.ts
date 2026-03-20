import { createRouter, createWebHashHistory } from 'vue-router'
// import HomeView from '../views/HomeView.vue'


const router = createRouter({
    // 使用 Hash 模式，URL 中会包含 #，例如：http://localhost/#/user/0
    history: createWebHashHistory(),
    routes: [
        {
            path: '/',
            redirect: "/user/info/0",
        },
        {
            path: '/login',
            name: 'login',
            component: import('@/views/layout/login.vue'),
        },
        {
            path: '/user',
            redirect: "/user/info/0",
            children: [
                {
                    path: 'info',
                    redirect: "/user/info/0",
                },
                {
                    path: 'info/:User_Id',
                    name: 'info',
                    component: import('@/views/user/user.vue'),
                },
                {
                    path: 'authority',
                    name: 'authority',
                    component: import('@/views/authority/authority.vue'),
                },
                {
                    path: 'authority_user',
                    name: 'authority_user',
                    component: import('@/views/authority/authority_user.vue'),
                },
                {
                    path: 'group',
                    name: 'group',
                    component: import('@/views/group/group.vue'),
                },
                {
                    path: 'group_user/:group_user__id',
                    name: 'group_user',
                    component: import('@/views/group/group_user.vue'),
                },
                {
                    path: 'user_account',
                    name: 'user_account',
                    component: import('@/views/user/user_account.vue'),
                },
            ]
        },

    ]
})

export default router
