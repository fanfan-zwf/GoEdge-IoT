// 全局扩展 Window
declare global {
  interface Window {
    APP_CONFIG?: {
      http_Front_url: string;
      config_service_url: string;
    };
  }
}

// 默认配置
const defaultConfig = {
  http_Front_url: "",
  config_service_url: "",
};

// 合并运行时配置
const config = {
  ...defaultConfig,
  ...(window.APP_CONFIG || {}),
};

// 给 Vue 全局添加 TS 类型
declare module '@vue/runtime-core' {
  interface ComponentCustomProperties {
    $config: typeof config;
  }
}

export default config;