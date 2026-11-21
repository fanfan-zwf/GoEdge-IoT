import vue from "@vitejs/plugin-vue";
import VueDevTools from "vite-plugin-vue-devtools";
import svgLoader from "vite-svg-loader";
import Icons from "unplugin-icons/vite";
import type { PluginOption } from "vite";
import { viteBuildInfo } from "./info.ts";
import vueJsx from "@vitejs/plugin-vue-jsx";
import tailwindcss from "@tailwindcss/vite";
import removeNoMatch from "vite-plugin-router-warn";
import { configCompressPlugin } from "./compress.ts";
import { visualizer } from "rollup-plugin-visualizer";
import removeConsole from "vite-plugin-remove-console";
import { codeInspectorPlugin } from "code-inspector-plugin";
import { vitePluginFakeServer } from "vite-plugin-fake-server";

export async function getPluginsList(
  VITE_CDN: boolean,
  VITE_COMPRESSION: ViteCompression,
  mode: string
): Promise<PluginOption[]> {
  const lifecycle = process.env.npm_lifecycle_event;
  // 仅开发（vite dev / serve）注入调试工具；生产构建剔除 devtools、代码审查器（源码路径泄露）等
  const isDev = mode === "development";
  return [
    tailwindcss(),
    vue({
      template: {
        compilerOptions: {
          isCustomElement: tag => tag === "deep-chat"
        }
      }
    }),
    isDev ? VueDevTools() : null,
    // jsx、tsx语法支持
    vueJsx(),
    /**
     * 在页面上按住组合键时，鼠标在页面移动即会在 DOM 上出现遮罩层并显示相关信息，点击一下将自动打开 IDE 并将光标定位到元素对应的代码位置
     * Mac 默认组合键 Option + Shift
     * Windows 默认组合键 Alt + Shift
     * 更多用法看 https://inspector.fe-dev.cn/guide/start.html
     * 仅开发环境启用，避免生产包内注入源码路径与点击跳转 IDE 的能力
     */
    isDev
      ? codeInspectorPlugin({
          bundler: "vite",
          hideConsole: true
        })
      : null,
    viteBuildInfo(),
    /**
     * 开发环境下移除非必要的vue-router动态路由警告No match found for location with path
     * 非必要具体看 https://github.com/vuejs/router/issues/521 和 https://github.com/vuejs/router/issues/359
     * vite-plugin-router-warn只在开发环境下启用，只处理vue-router文件并且只在服务启动或重启时运行一次，性能消耗可忽略不计
     */
    removeNoMatch(),
    // mock支持
    vitePluginFakeServer({
      logger: false,
      include: "mock",
      infixName: false,
      enableProd: true
    }),
    // svg组件化支持
    svgLoader(),
    // 自动按需加载图标
    Icons({
      compiler: "vue3",
      scale: 1
    }),
    VITE_CDN ? (await import("./cdn.ts")).cdn : null,
    configCompressPlugin(VITE_COMPRESSION),
    // 线上环境删除console
    removeConsole({ external: ["src/assets/iconfont/iconfont.js"] }),
    // 打包分析
    lifecycle === "report"
      ? visualizer({ open: true, brotliSize: true, filename: "report.html" })
      : (null as any)
  ];
}
