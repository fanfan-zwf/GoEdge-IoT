import { $t } from "@/plugins/i18n";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/",
  name: "Home",
  component: Layout,
  redirect: "/home",
  meta: {
    icon: "ep/home-filled",
    title: $t("menus.pureHome"),
    rank: 0
  },
  children: [
    {
      path: "/home",
      name: "HomePage",
      component: () => import("@/views/home/index.vue"),
      meta: {
        title: $t("menus.pureHome"),
        icon: "ep/home-filled"
      }
    }
  ]
} satisfies RouteConfigsTable;
