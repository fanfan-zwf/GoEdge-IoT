import { $t } from "@/plugins/i18n";
import { systemConfig } from "@/router/enums";
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/system",
  name: "System",
  component: Layout,
  redirect: "/system/config",
  meta: {
    icon: "ep/setting",
    title: $t("menus.pureSystemManage"),
    rank: systemConfig
  },
  children: [
    {
      path: "/system/config",
      name: "SystemConfig",
      component: () => import("@/views/system-config/index.vue"),
      meta: {
        title: $t("menus.pureSystemConfig"),
        icon: "ep/tools"
      }
    },
    {
      path: "/system/info",
      name: "SystemInfo",
      component: () => import("@/views/system-info/index.vue"),
      meta: {
        title: $t("menus.pureSystemInfo"),
        icon: "ep/data-line"
      }
    },
    {
      path: "/system/user",
      name: "SystemUser",
      component: () => import("@/views/user/index.vue"),
      meta: {
        title: $t("menus.pureUser"),
        icon: "ep/user"
      }
    },
    {
      path: "/system/apikey",
      name: "SystemApiKey",
      component: () => import("@/views/apikey/index.vue"),
      meta: {
        title: $t("menus.pureApiKey"),
        icon: "ep/key"
      }
    },
    {
      path: "/system/rbac",
      name: "SystemRbac",
      component: () => import("@/views/rbac/index.vue"),
      meta: {
        title: $t("menus.pureRbac"),
        icon: "ep/lock",
        // RBAC 管理页可见性：登录 roles 含 "*"（旧硬编码超管）或 "super_admin"（引导全权限角色）才展示；
        // 未分配 RBAC 相关角色的普通用户不展示此菜单（后端仍是 API 级鉴权的唯一权威，会返回 403）
        roles: ["*", "super_admin"]
      }
    }
  ]
} satisfies RouteConfigsTable;
