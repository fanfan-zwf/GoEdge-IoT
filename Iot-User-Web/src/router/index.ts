import { createRouter, createWebHashHistory } from 'vue-router'
// import HomeView from '../views/HomeView.vue'


// 路由画面
import login from '@/views/layout/login.vue'
import user from '@/views/user/user.vue'
import authority from '@/views/authority/authority.vue'
import authority_user from '@/views/authority/authority_user.vue'
import group from '@/views/group/group.vue'
import group_user from '@/views/group/group_user.vue'
import user_account from '@/views/user/user_account.vue'


const router = createRouter({
    history: createWebHashHistory(),
    routes: [
        {
            path: '/',
            redirect: "/user/0",
        },
        {
            path: '/user', 
            children: [
                {
                    path: 'login',
                    name: 'login',
                    component: login,
                },
                {
                    path: 'info/:User_Id',
                    name: 'user',
                    component: user,
                },
                {
                    path: 'authority',
                    name: 'authority',
                    component: authority,
                },
                {
                    path: 'authority_user',
                    name: 'authority_user',
                    component: authority_user,
                },
                {
                    path: 'group',
                    name: 'group',
                    component: group,
                },
                {
                    path: 'group_user/:group_user__id',
                    name: 'group_user',
                    component: group_user,
                },
                {
                    path: 'user_account',
                    name: 'user_account',
                    component: user_account,
                },
            ],
        },

    ]
})

export default router
