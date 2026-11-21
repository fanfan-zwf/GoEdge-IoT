<template>
  <el-form :model="driveForm" label-position="top" class="drive-form">
    <el-divider content-position="left">{{ t("modbus.driveConfig") }}</el-divider>
    <el-row :gutter="16">
      <el-col :span="8">
        <el-form-item :label="t('modbus.driveAddress')">
          <el-input v-model="driveForm.Address" placeholder="192.168.1.1:502" />
        </el-form-item>
      </el-col>
      <el-col :span="8">
        <el-form-item :label="t('modbus.retryTimeout')">
          <el-input v-model="driveForm.Retry_timeout" placeholder="3s" />
        </el-form-item>
      </el-col>
      <el-col :span="8">
        <el-form-item :label="t('modbus.connectTimeout')">
          <el-input v-model="driveForm.Connect_timeout" placeholder="10s" />
        </el-form-item>
      </el-col>
    </el-row>
    <el-row :gutter="16">
      <el-col :span="8">
        <el-form-item :label="t('modbus.responseTimeout')">
          <el-input v-model="driveForm.Response_timeout" placeholder="10s" />
        </el-form-item>
      </el-col>
      <el-col :span="8">
        <el-form-item :label="t('modbus.delayBetweenPolls')">
          <el-input v-model="driveForm.Delay_between_polls" placeholder="0s" />
        </el-form-item>
      </el-col>
      <el-col :span="8">
        <el-form-item :label="t('modbus.packetMax')">
          <el-input-number v-model="driveForm.Packet_max" :min="1" :max="250" :controls="false" placeholder="120" style="width: 100%" />
        </el-form-item>
      </el-col>
    </el-row>
  </el-form>
</template>

<script setup lang="ts">
import { watch } from "vue";
import { useI18n } from "vue-i18n";
import type { ModbusConfig } from "@/views/ioDrive/types";
import { MODBUS_CONFIG_DEFAULTS, parseDriveConfigJson, stringifyDriveConfig } from "@/views/ioDrive/types";

// driveForm：结构化驱动配置；config：父组件的 Config 文本（JSON），两者由本组件保持双向同步
const driveForm = defineModel<ModbusConfig>({ required: true });
const config = defineModel<string>("config", { default: "" });

const { t } = useI18n();

/** 结构化字段 → JSON 文本；内容一致时跳过，省掉一次无谓的父子来回 */
const writeConfig = () => {
  const text = stringifyDriveConfig(driveForm.value);
  if (text !== config.value) config.value = text;
};

/** JSON 文本 → 结构化字段；文本为空或无法解析（新增、切换驱动）时反向生成默认配置 */
const readConfig = (raw?: string) => {
  const parsed = parseDriveConfigJson(raw ?? "", MODBUS_CONFIG_DEFAULTS);
  if (parsed) Object.assign(driveForm.value, parsed);
  else writeConfig();
};

// 首次挂载（对话框每次打开都会重建本组件）：以父级 Config 回填字段，无配置时生成默认 JSON
readConfig(config.value);

// 父级在外部改动 Config（编辑回填、切换驱动清空）→ 同步到结构化字段
watch(config, readConfig);

// 本组件字段改动 → 实时写回父级 Config
watch(driveForm, writeConfig, { deep: true });
</script>

<style scoped>
.drive-form {
  padding: 10px;
}

.drive-form :deep(.el-form-item) {
  margin-bottom: 16px;
}

.drive-form :deep(.el-form-item__label) {
  padding-bottom: 4px;
}
</style>
