import "@/utils/sso";
import Cookies from "js-cookie";
import { getConfig } from "@/config";
import NProgress from "@/utils/progress";
import { message } from "@/utils/message";
import { transformI18n } from "@/plugins/i18n";
import { buildHierarchyTree } from "@/utils/tree";
import remainingRouter from "./modules/remaining";
import { useMultiTagsStoreHook } from "@/store/modules/multiTags";
import { usePermissionStoreHook } from "@/store/modules/permission";
import { isUrl, openLink, cloneDeep, isAllEmpty, storageLocal } from "@pureadmin/utils";
import {
  ascending,
  getTopMenu,
  initRouter,
  getHistoryMode,
  findRouteByPath,
  handleAliveRoute,
  formatTwoStageRoutes,
  formatFlatteningRoutes
} from "./utils";
import { type Router, type RouteRecordRaw, type RouteComponent, createRouter } from "vue-router";
import { type DataInfo, userKey, multipleTabsKey, getToken } from "@/utils/auth";

/** 自动导入全部静态路由，无需再手动引入！匹配 src/router/modules 目录（任何嵌套级别）中具有 .ts 扩展名的所有文件，除了 remaining.ts 文件
 * 如何匹配所有文件请看：https://github.com/mrmlnc/fast-glob#basic-syntax
 * 如何排除文件请看：https://cn.vitejs.dev/guide/features.html#negative-patterns
 */
const modules: Record<string, any> = import.meta.glob(["./modules/**/*.ts", "!./modules/**/remaining.ts"], {
  eager: true
});

/** 原始静态路由（未做任何处理） */
const routes = [];

Object.keys(modules).forEach(key => {
  routes.push(modules[key].default);
});

/** 导出处理后的静态路由（三级及以上的路由全部拍成二级） */
export const constantRoutes: Array<RouteRecordRaw> = formatTwoStageRoutes(
  formatFlatteningRoutes(buildHierarchyTree(ascending(routes.flat(Infinity))))
);

/** 初始的静态路由，用于退出登录时重置路由 */
const initConstantRoutes: Array<RouteRecordRaw> = cloneDeep(constantRoutes);

/** 用于渲染菜单，保持原始层级 */
export const constantMenus: Array<RouteComponent> = ascending(routes.flat(Infinity)).concat(...remainingRouter);

/** 不参与菜单的路由 */
export const remainingPaths = Object.keys(remainingRouter).map(v => {
  return remainingRouter[v].path;
});

/** 创建路由实例 */
export const router: Router = createRouter({
  history: getHistoryMode(import.meta.env.VITE_ROUTER_HISTORY),
  routes: constantRoutes.concat(...(remainingRouter as any)),
  strict: true,
  scrollBehavior(to, from, savedPosition) {
    return new Promise(resolve => {
      if (savedPosition) {
        return savedPosition;
      } else {
        if (from.meta.saveSrollTop) {
          const top: number = document.documentElement.scrollTop || document.body.scrollTop;
          resolve({ left: 0, top });
        }
      }
    });
  }
});

/** 记录已经加载的页面路径 */
const loadedPaths = new Set<string>();

/** 重置已加载页面记录 */
export function resetLoadedPaths() {
  loadedPaths.clear();
}

/** 重置路由 */
export function resetRouter() {
  router.clearRoutes();
  for (const route of initConstantRoutes.concat(...(remainingRouter as any))) {
    router.addRoute(route);
  }
  router.options.routes = formatTwoStageRoutes(
    formatFlatteningRoutes(buildHierarchyTree(ascending(routes.flat(Infinity))))
  );
  usePermissionStoreHook().clearAllCachePage();
  resetLoadedPaths();
}

/** 路由白名单 */
const whiteList = ["/login"];

router.beforeEach((to: ToRouteType, _from) => {
  to.meta.loaded = loadedPaths.has(to.path);

  if (!to.meta.loaded) {
    NProgress.start();
  }

  if (to.meta?.keepAlive) {
    handleAliveRoute(to, "add");
    // 页面整体刷新和点击标签页刷新
    if (_from.name === undefined || _from.name === "Redirect") {
      handleAliveRoute(to);
    }
  }
  const userInfo = storageLocal().getItem<DataInfo<number>>(userKey);
  const externalLink = isUrl(to?.name as string);
  if (!externalLink) {
    to.matched.some(item => {
      if (!item.meta.title) return "";
      const Title = getConfig().Title;
      if (Title) document.title = `${transformI18n(item.meta.title)} | ${Title}`;
      else document.title = transformI18n(item.meta.title);
    });
  }
  /** 如果已经登录并存在登录信息后不能跳转到路由白名单，而是继续保持在当前页面 */
  function toCorrectRoute() {
    return whiteList.includes(to.fullPath) ? _from.fullPath : undefined;
  }
  // 未登录：仅允许访问白名单（/login），否则重定向到登录页
  // 兼容旧版“自动假登录”遗留的无效 token（accessToken=auto-login），一并视为未登录
  const accessToken = getToken()?.accessToken;
  if (!Cookies.get(multipleTabsKey) || !userInfo || !accessToken || accessToken === "auto-login") {
    return whiteList.includes(to.path) ? undefined : "/login";
  }
  if (_from?.name) {
    // name为超链接
    if (externalLink) {
      openLink(to?.name as string);
      NProgress.done();
      return false;
    } else {
      return toCorrectRoute();
    }
  } else {
    // 刷新
    if (usePermissionStoreHook().wholeMenus.length === 0 && to.path !== "/login") {
      initRouter().then((router: Router) => {
        if (!useMultiTagsStoreHook().getMultiTagsCache) {
          const { path } = to;
          const route = findRouteByPath(path, router.options.routes[0].children);
          getTopMenu(true);
          // query、params模式路由传参数的标签页不在此处处理
          if (route && route.meta?.title) {
            if (isAllEmpty(route.parentId) && route.meta?.backstage) {
              // 此处为动态顶级路由（目录）
              const { path, name, meta } = route.children[0];
              useMultiTagsStoreHook().handleTags("push", {
                path,
                name,
                meta
              });
            } else {
              const { path, name, meta } = route;
              useMultiTagsStoreHook().handleTags("push", {
                path,
                name,
                meta
              });
            }
          }
        }
        // 确保动态路由完全加入路由列表并且不影响静态路由
        if (isAllEmpty(to.name)) router.push(to.fullPath);
      });
    }
    return toCorrectRoute();
  }
});

router.afterEach(to => {
  loadedPaths.add(to.path);
  NProgress.done();
});

/** 全局路由错误处理：捕获导航异常（如 VUE_ROUTER_R0010 缺参/导航中断、动态 chunk 加载失败等），
 * 收尾进度条并上报 + 提示，避免异常静默留在控制台。 */
router.onError(error => {
  NProgress.done();
  console.error("[router] navigation error:", error);
  const msg = String((error as any)?.message ?? error ?? "");
  // 重新部署后旧分包 hash 失效导致的动态导入失败：提示用户刷新获取最新资源
  if (/dynamically imported module|Importing a module script failed|Failed to fetch dynamically|loading dynamically imported module/i.test(msg)) {
    message("页面资源已更新，请刷新后重试", { type: "error" });
  } else {
    message("页面跳转异常，请重试", { type: "error" });
  }
});

export default router;
