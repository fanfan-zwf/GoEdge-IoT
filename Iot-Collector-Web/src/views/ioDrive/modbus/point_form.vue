<template>
  <el-form :model="pointForm" label-position="top" class="point-form">
    <el-divider content-position="left">{{ t("modbus.pointConfig") }}</el-divider>
    <el-row :gutter="16">
      <el-col :span="6">
        <el-form-item :label="t('modbus.slaveId')">
          <el-input-number v-model="pointForm.SlaveID" :min="0" :max="255" :controls="false" style="width: 100%" />
        </el-form-item>
      </el-col>
      <el-col :span="6">
        <el-form-item :label="t('modbus.function')">
          <el-select v-model="pointForm.Function" style="width: 100%">
            <el-option v-for="item in MODBUS_FUNCTION_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
      </el-col>
      <el-col :span="6">
        <el-form-item :label="t('modbus.address')">
          <el-input-number v-model="pointForm.Address" :min="0" :max="65535" :controls="false" style="width: 100%" />
        </el-form-item>
      </el-col>
      <!-- 位偏移: 仅 bool 类型显示 -->
      <el-col v-if="pointForm.DataType === 'bool'" :span="6">
        <el-form-item :label="t('s7.childAddress')">
          <el-input-number v-model="pointForm.Child_Address" :min="0" :max="15" :controls="false" style="width: 100%" />
        </el-form-item>
      </el-col>
    </el-row>
    <el-row :gutter="16">
      <el-col :span="6">
        <el-form-item :label="t('modbus.dataType')">
          <el-select v-model="pointForm.DataType" style="width: 100%">
            <el-option v-for="item in MODBUS_DATA_TYPE_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
      </el-col>
      <el-col :span="6">
        <el-form-item :label="t('modbus.byteOrder')">
          <el-select v-model="pointForm.ByteOrder" style="width: 100%">
            <el-option v-for="item in MODBUS_BYTE_ORDER_OPTIONS" :key="item.value" :label="item.label" :value="item.value" />
          </el-select>
        </el-form-item>
      </el-col>
    </el-row>
  </el-form>
</template>

<script setup lang="ts">
import { computed, watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ModbusPointConfig } from "@/views/ioDrive/types";
import { MODBUS_FUNCTION_OPTIONS, MODBUS_DATA_TYPE_OPTIONS, MODBUS_BYTE_ORDER_OPTIONS } from "@/views/ioDrive/types";

const pointForm = defineModel<ModbusPointConfig>({ required: true });
// config：父组件的 Config 文本，与 pointForm 由本组件保持双向同步
const config = defineModel<string>("config", { default: "" });

const { t } = useI18n();

// 生成 Modbus 点位配置字符串: 从机地址:功能码:寄存器地址[.位偏移]:字节序:数据类型
// 位偏移仅在 bool 类型时拼接（对应后端正则的可选子地址）
const modbusAddress = computed(() => {
  const f = pointForm.value;
  const addr = f.DataType === "bool" ? `${f.Address}.${f.Child_Address}` : `${f.Address}`;
  return `${f.SlaveID}:${f.Function}:${addr}:${f.ByteOrder}:${f.DataType}`;
});

// ==================== Config 文本 ⇄ 结构化字段 双向同步 ====================

// 与后端 Regex_Modbus_Point 同源：从机地址:功能码:寄存器地址[.位偏移]:字节序:数据类型
const MODBUS_POINT_RE = /^(\d+):([1-4]):(\d+)(?:\.(\d+))?:([\w]+):([\w]+)$/;

/** 解析配置字符串为结构化字段；字节序/数据类型不在候选表内返回 null，保留原文交给表单校验 */
const parseConfig = (raw: string): ModbusPointConfig | null => {
  const m = (raw ?? "").trim().match(MODBUS_POINT_RE);
  if (!m) return null;
  const slaveId = Number(m[1]);
  const address = Number(m[3]);
  const byteOrder = m[5].toUpperCase();
  const dataType = m[6].toLowerCase();
  if (slaveId > 255 || address > 65535) return null;
  if (!MODBUS_BYTE_ORDER_OPTIONS.some(o => o.value === byteOrder)) return null;
  if (!MODBUS_DATA_TYPE_OPTIONS.some(o => o.value === dataType)) return null;
  return {
    SlaveID: slaveId,
    Function: Number(m[2]),
    Address: address,
    // 位偏移仅 bool 有效，其余类型一律归零
    Child_Address: dataType === "bool" ? Number(m[4] ?? 0) : 0,
    ByteOrder: byteOrder,
    DataType: dataType
  };
};

/** 结构化字段 → Config 文本；内容一致时跳过，省掉一次无谓的父子来回 */
const writeConfig = () => {
  const text = modbusAddress.value;
  if (text !== config.value) config.value = text;
};

/** Config 文本 → 结构化字段；文本为空（新增、切换驱动）时反向生成配置 */
const readConfig = (raw?: string) => {
  const text = (raw ?? "").trim();
  if (!text) {
    writeConfig();
    return;
  }
  const parsed = parseConfig(text);
  if (parsed) Object.assign(pointForm.value, parsed);
};

// 首次挂载（对话框每次打开都会重建本组件）：以父级 Config 回填字段，无配置时生成默认值
readConfig(config.value);

// 父级在外部改动 Config（编辑回填、切换驱动清空）→ 同步到结构化字段
watch(config, readConfig);

// 本组件字段改动 → 实时写回父级 Config
watch(modbusAddress, writeConfig);
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
