<template>
  <el-form :model="pointForm" label-position="top" class="point-form">
    <el-divider content-position="left">{{ t("s7.pointConfig") }}</el-divider>
    <el-row :gutter="16">
      <el-col :span="6">
        <el-form-item :label="t('s7.area')">
          <el-select v-model="pointForm.Area" style="width: 100%" @change="handleAreaChange">
            <el-option v-for="item in S7_AREA_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
      </el-col>
      <!-- 只有 DB 区才显示 DB号 -->
      <el-col v-if="pointForm.Area === 0x84" :span="6">
        <el-form-item :label="t('s7.dbNumber')">
          <el-input-number v-model="pointForm.DBNumber" :min="0" :controls="false" style="width: 100%" />
        </el-form-item>
      </el-col>
      <el-col :span="6">
        <el-form-item :label="t('s7.start')">
          <el-input-number v-model="pointForm.Start" :min="0" :controls="false" style="width: 100%" />
        </el-form-item>
      </el-col>
      <el-col :span="6">
        <el-form-item :label="t('s7.dataType')">
          <el-select v-model="pointForm.Type" style="width: 100%" @change="handleTypeChange">
            <el-option
              v-for="item in S7_DATA_TYPE_OPTIONS"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
      </el-col>
      <!-- 只有 Bool 采集类型才显示位偏移 -->
      <el-col v-if="pointForm.Type === 1" :span="6">
        <el-form-item :label="t('s7.childAddress')">
          <el-input-number v-model="pointForm.Child_Address" :min="0" :max="7" :controls="false" style="width: 100%" />
        </el-form-item>
      </el-col>
    </el-row>
  </el-form>
</template>

<script setup lang="ts">
import { computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { SiemensPointConfig } from "@/views/ioDrive/types";
import { S7_AREA_OPTIONS, S7_DATA_TYPE_OPTIONS } from "@/views/ioDrive/types";

// pointForm：可视化结构化字段；config：父组件的 Config 文本，两者由本组件保持双向同步
const pointForm = defineModel<SiemensPointConfig>({ required: true });
const config = defineModel<string>("config", { default: "" });

const { t } = useI18n();

// 根据 Area 获取区域前缀
const getAreaPrefix = (area: number): string => {
  switch (area) {
    case 0x81:
      return "I";
    case 0x82:
      return "Q";
    case 0x83:
      return "M";
    case 0x84:
      return "DB";
    default:
      return "DB";
  }
};

// 采集类型代码 → 地址宽度后缀 / TIA 名称（依据 S7_DATA_TYPE_OPTIONS 表）
const findTypeMeta = (code: number) => {
  return S7_DATA_TYPE_OPTIONS.find(o => o.value === code) || S7_DATA_TYPE_OPTIONS[0];
};

// 生成 S7 地址字符串：形如 "DB1.DBD0 Real" / "M10.0 Bool"
// 后缀类型名与后端 new.go Point_Config_Switch 的 "addr [TIA类型名]" 语义完全对齐
const s7Address = computed(() => {
  const area = pointForm.value.Area;
  const areaPrefix = getAreaPrefix(area);
  const meta = findTypeMeta(pointForm.value.Type);
  const typeSuffix = meta.suffix;
  const start = pointForm.value.Start;
  const childAddr = pointForm.value.Child_Address;

  let addr: string;
  if (area === 0x84) {
    // DB 区: DB1.DBX0.0, DB1.DBB2, DB1.DBW4, DB1.DBD8, DB1.DBL16
    const dbNum = pointForm.value.DBNumber;
    if (typeSuffix === "X") {
      addr = `DB${dbNum}.DB${typeSuffix}${start}.${childAddr}`;
    } else {
      addr = `DB${dbNum}.DB${typeSuffix}${start}`;
    }
  } else {
    // I/Q/M 区: I0.0, IB2, IW4, ID8, IL16
    if (typeSuffix === "X") {
      addr = `${areaPrefix}${start}.${childAddr}`;
    } else {
      addr = `${areaPrefix}${typeSuffix}${start}`;
    }
  }

  // 拼接显式 TIA 类型名，避免后端按后缀推断出错误的无符号/有符号类型（如 D 后缀默认 UDInt 但实际是 Real/DInt）
  return `${addr} ${meta.tia}`;
});

// 监听存储区变化，非 DB 时重置 DB号
const handleAreaChange = (value: number) => {
  if (value !== 0x84) {
    pointForm.value.DBNumber = 0;
  }
};

// 监听采集类型变化，非 Bool 时重置位偏移
const handleTypeChange = (value: number) => {
  if (value !== 1) {
    pointForm.value.Child_Address = 0;
  }
};

// 兼容：外部若通过 defineModel 修改 Type，同样保证位偏移清理
watch(
  () => pointForm.value.Type,
  newVal => {
    if (newVal !== 1) {
      pointForm.value.Child_Address = 0;
    }
  }
);

// ==================== Config 文本 ⇄ 结构化字段 双向同步 ====================

// TIA 类型名（小写）→ 采集类型代码，代码表见 types.ts 的 S7_DATA_TYPE_OPTIONS
const TIA_TYPE_CODE: Record<string, number> = Object.fromEntries(
  S7_DATA_TYPE_OPTIONS.map(o => [o.tia.toLowerCase(), o.value])
);
// 别名 → 规范 TIA 名，与后端 new.go 的 s7TypeMap 保持一致
const TIA_ALIAS: Record<string, string> = { byte: "USInt", word: "UInt", dword: "UDInt", lword: "ULInt" };
const typeByTiaName = (name: string): number | undefined =>
  TIA_TYPE_CODE[(TIA_ALIAS[name.toLowerCase()] ?? name).toLowerCase()];
// 地址宽度后缀 → 默认 TIA 类型，与后端 inferS7TypeBySuffix 一致
const SUFFIX_TIA: Record<string, string> = { X: "Bool", B: "USInt", W: "UInt", D: "UDInt", L: "ULInt" };
// 区域前缀 → 存储区代码
const PREFIX_AREA: Record<string, number> = { I: 0x81, Q: 0x82, M: 0x83, DB: 0x84 };
// 与后端 Regex_S7_Point_Config 同源
const S7_ADDR_RE = /^(?:(DB)(\d+)\.(DB[XBWDL])|([IQM])([XBWDL]?))(\d+)(?:\.(\d+))?$/;

/** 解析 "<地址> [TIA类型名]" 为结构化字段；无法识别返回 null，由调用方保留原文交给表单校验 */
const parseAddress = (raw: string): SiemensPointConfig | null => {
  const text = (raw ?? "").trim();
  if (!text) return null;
  const sep = text.search(/\s/);
  const addr = (sep < 0 ? text : text.slice(0, sep)).toUpperCase();
  const typeName = sep < 0 ? "" : text.slice(sep + 1).trim();

  const m = addr.match(S7_ADDR_RE);
  if (!m) return null;

  const isDb = m[1] === "DB";
  const suffix = (isDb ? m[3].slice(2) : m[5]) || "";
  // 采集类型优先级：显式类型名 → 地址宽度后缀 → 位号（M10.3 这类无后缀的位寻址）→ 保持当前值
  const type =
    typeByTiaName(typeName) ??
    (suffix ? typeByTiaName(SUFFIX_TIA[suffix] ?? "") : undefined) ??
    (m[7] !== undefined ? TIA_TYPE_CODE["bool"] : undefined) ??
    pointForm.value.Type;

  return {
    Area: isDb ? 0x84 : (PREFIX_AREA[m[4]] ?? 0x84),
    DBNumber: isDb ? Number(m[2]) : 0,
    Start: Number(m[6]),
    // 位偏移仅 Bool 有效，其余区域一律归零，避免回填出无法表达的字段
    Child_Address: type === 1 ? Number(m[7] ?? 0) : 0,
    Type: type
  };
};

/** 结构化字段 → Config 文本；内容一致时跳过，省掉一次无谓的父子来回 */
const writeConfig = () => {
  const text = s7Address.value;
  if (text !== config.value) config.value = text;
};

/** Config 文本 → 结构化字段；文本为空（新增、切换驱动）时反向生成地址 */
const readConfig = (raw?: string) => {
  const text = (raw ?? "").trim();
  if (!text) {
    writeConfig();
    return;
  }
  const parsed = parseAddress(text);
  if (parsed) Object.assign(pointForm.value, parsed);
};

// 首次挂载（对话框每次打开都会重建本组件）：以父级 Config 回填字段，无配置时生成默认地址
readConfig(config.value);

// 父级在外部改动 Config（编辑回填、切换驱动清空）→ 同步到结构化字段
watch(config, readConfig);

// 本组件字段改动（含区域、类型的联动重置）→ 实时写回父级 Config
watch(s7Address, writeConfig);
</script>

<style scoped>
.point-form {
  padding: 10px;
}

.point-form :deep(.el-form-item) {
  margin-bottom: 16px;
}

.point-form :deep(.el-form-item__label) {
  padding-bottom: 4px;
}
</style>
