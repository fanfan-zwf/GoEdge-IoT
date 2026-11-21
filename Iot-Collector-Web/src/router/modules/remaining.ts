import { $t } from "@/plugins/i18n";
const Layout = () => import("@/layout/index.vue");

export default [
  {
    path: "/login",
    name: "Login",
    component: () => import("@/views/login/index.vue"),
    meta: {
      title: $t("menus.pureLogin"),
      showLink: false
    }
  },
  {
    path: "/redirect",
    component: Layout,
    meta: {
      title: $t("status.pureLoad"),
      showLink: false
    },
    children: [
      {
        path: "/redirect/:path(.*)",
        name: "Redirect",
        component: () => import("@/layout/redirect.vue")
      }
    ]
  },
  // 历史数据页面（隐藏菜单，通过历史配置页跳转）
  {
    path: "/history-data/:pointId",
    component: Layout,
    meta: {
      title: $t("menus.pureHistoryData"),
      showLink: false
    },
    children: [
      {
        path: "",
        name: "HistoryData",
        component: () => import("@/views/history-data/index.vue"),
        meta: {
          title: $t("menus.pureHistoryData"),
          showLink: true
        }
      }
    ]
  },
  // 报警数据页面（隐藏菜单，通过报警配置页跳转）
  {
    path: "/alarm-data/:alarmId",
    component: Layout,
    meta: {
      title: $t("menus.pureAlarmData"),
      showLink: false
    },
    children: [
      {
        path: "",
        name: "AlarmData",
        component: () => import("@/views/alarm-data/index.vue"),
        meta: {
          title: $t("menus.pureAlarmData"),
          showLink: true
        }
      }
    ]
  }
] satisfies Array<RouteConfigsTable>;
