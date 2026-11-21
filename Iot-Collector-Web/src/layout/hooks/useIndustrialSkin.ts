import { computed } from "vue";
import { useGlobal } from "@pureadmin/utils";
import { useDataThemeChange } from "./useDataThemeChange";

export type IndustrialSkinId =
  | "industrialConsole"
  | "industrialBlueprint"
  | "industrialPanel"
  | "classic";

export type IndustrialSkinMeta = {
  id: IndustrialSkinId;
  color: string;
  labelKey: string;
  tag: string;
  forceDark: boolean;
};

/** 三套工业皮肤的元数据（与 useDataThemeChange.themeColors 中新增项一一对应） */
export const INDUSTRIAL_SKINS: IndustrialSkinMeta[] = [
  {
    id: "industrialConsole",
    color: "#4dd0e1",
    labelKey: "skin.console",
    tag: "// DEEP CONSOLE · TERMINAL v1.0",
    forceDark: true
  },
  {
    id: "industrialBlueprint",
    color: "#1e6fd9",
    labelKey: "skin.blueprint",
    tag: "// DWG: PLC-Arch-01 / REV: A",
    forceDark: false
  },
  {
    id: "industrialPanel",
    color: "#ff8a3d",
    labelKey: "skin.panel",
    tag: "// HMI-1024 · IP65 · 24VDC",
    forceDark: true
  }
];

const SKIN_IDS = new Set<string>(
  INDUSTRIAL_SKINS.map(s => s.id as string).concat(["classic"])
);

/**
 * 工业皮肤 composable：从当前布局主题派生皮肤 id，并提供切换函数
 */
export function useIndustrialSkin() {
  const { $storage } = useGlobal<GlobalPropertiesApi>();
  const { setLayoutThemeColor, dataThemeChange, dataTheme } =
    useDataThemeChange();

  /** 当前皮肤 id（非工业主题统一归为 classic） */
  const skin = computed<IndustrialSkinId>(() => {
    const theme = $storage?.layout?.theme;
    if (theme && SKIN_IDS.has(theme)) return theme as IndustrialSkinId;
    return "classic";
  });

  /** 是否处于工业皮肤模式 */
  const isIndustrial = computed(() => skin.value !== "classic");

  /** 当前皮肤元数据（classic 时为 undefined） */
  const current = computed<IndustrialSkinMeta | undefined>(() =>
    INDUSTRIAL_SKINS.find(s => s.id === skin.value)
  );

  /** 切换皮肤；同时按皮肤语义设置 darkMode */
  function setSkin(id: IndustrialSkinId) {
    if (id === "classic") {
      // 回退到 classic：保留当前 darkMode 由用户自行控制
      setLayoutThemeColor("light");
      return;
    }
    const meta = INDUSTRIAL_SKINS.find(s => s.id === id);
    if (!meta) return;
    // 先设置 data-theme 属性
    setLayoutThemeColor(id);
    // 再按 forceDark 同步 darkMode
    dataTheme.value = meta.forceDark;
    dataThemeChange(meta.forceDark ? "dark" : "light");
  }

  return {
    skin,
    isIndustrial,
    current,
    setSkin,
    skins: INDUSTRIAL_SKINS
  };
}
