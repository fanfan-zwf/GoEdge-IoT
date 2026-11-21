import { $t } from "@/plugins/i18n";
import { drive } from "@/router/enums";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/point-manage",
  name: "PointManage",
  component: Layout,
  redirect: "/point-manage/drive",
  meta: {
    icon: "ep/management",
    title: $t("menus.purePointManage"),
    rank: drive
  },
  children: [
    {
      path: "/point-manage/drive",
      name: "Drive",
      component: () => import("@/views/drive/index.vue"),
      meta: {
        title: $t("menus.pureDrive"),
        icon: "ep/monitor"
      }
    },
    {
      path: "/point-manage/point",
      name: "Point",
      component: () => import("@/views/point/index.vue"),
      meta: {
        title: $t("menus.purePoint"),
        icon: "ep/connection"
      }
    },
    {
      path: "/point-manage/alarm",
      name: "Alarm",
      component: () => import("@/views/alarm/index.vue"),
      meta: {
        title: $t("menus.pureAlarm"),
        icon: "ep/bell"
      }
    },
    {
      path: "/point-manage/history",
      name: "History",
      component: () => import("@/views/history/index.vue"),
      meta: {
        title: $t("menus.pureHistory"),
        icon: "ep/clock"
      }
    }
  ]
} satisfies RouteConfigsTable;
