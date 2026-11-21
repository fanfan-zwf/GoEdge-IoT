<template>
  <el-form :model="driveForm" label-position="top" class="drive-form">
    <el-divider content-position="left">{{ t("s7.driveConfig") }}</el-divider>
    <el-row :gutter="16">
      <el-col :span="8">
        <el-form-item :label="t('s7.ip')">
          <el-input v-model="driveForm.Address" placeholder="192.168.1.1" />
        </el-form-item>
      </el-col>
      <el-col :span="8">
        <el-form-item :label="t('s7.rack')">
          <el-input-number v-model="driveForm.Rack" :min="0" :max="7" :controls="false" placeholder="0" style="width: 100%" />
        </el-form-item>
      </el-col>
      <el-col :span="8">
        <el-form-item :label="t('s7.slot')">
          <el-input-number v-model="driveForm.Slot" :min="0" :max="31" :controls="false" placeholder="2" style="width: 100%" />
        </el-form-item>
      </el-col>
    </el-row>
    <el-row :gutter="16">
      <el-col :span="8">
        <el-form-item :label="t('s7.retryTimeout')">
          <el-input v-model="driveForm.Retry_timeout" placeholder="3s" />
        </el-form-item>
      </el-col>
      <el-col :span="8">
        <el-form-item :label="t('s7.connectTimeout')">
          <el-input v-model="driveForm.Connect_timeout" placeholder="10s" />
        </el-form-item>
      </el-col>
      <el-col :span="8">
        <el-form-item :label="t('s7.responseTimeout')">
          <el-input v-model="driveForm.Response_timeout" placeholder="10s" />
        </el-form-item>
      </el-col>
    </el-row>
    <el-row :gutter="16">
      <el-col :span="8">
        <el-form-item :label="t('s7.delayBetweenPolls')">
          <el-input v-model="driveForm.Delay_between_polls" placeholder="0s" />
        </el-form-item>
      </el-col>
      <el-col :span="8">
        <el-form-item :label="t('s7.connectionType')">
          <el-select v-model="driveForm.ConnectionType" style="width: 100%">
            <el-option :label="t('s7.connPG')" :value="1" />
            <el-option :label="t('s7.connOP')" :value="2" />
            <el-option :label="t('s7.connBasic')" :value="3" />
          </el-select>
        </el-form-item>
      </el-col>
      <el-col :span="8">
        <el-form-item :label="t('s7.maxReconnectInterval')">
          <el-input v-model="driveForm.Max_reconnect_interval" placeholder="30s" />
        </el-form-item>
      </el-col>
    </el-row>
    <el-row :gutter="16">
      <el-col :span="12">
        <el-form-item :label="t('s7.localTsap')">
          <el-input-number v-model="driveForm.Local_TSAP" :min="0" :max="65535" :controls="false" placeholder="0" style="width: 100%" />
        </el-form-item>
      </el-col>
      <el-col :span="12">
        <el-form-item :label="t('s7.remoteTsap')">
          <el-input-number v-model="driveForm.Remote_TSAP" :min="0" :max="65535" :controls="false" placeholder="0" style="width: 100%" />
        </el-form-item>
      </el-col>
    </el-row>
  </el-form>
</template>

<script setup lang="ts">
import { watch } from "vue";
import { useI18n } from "vue-i18n";
import type { SiemensConfig } from "@/views/ioDrive/types";
import { SIEMENS_CONFIG_DEFAULTS, parseDriveConfigJson, stringifyDriveConfig } from "@/views/ioDrive/types";

// driveForm：结构化驱动配置；config：父组件的 Config 文本（JSON），两者由本组件保持双向同步
const driveForm = defineModel<SiemensConfig>({ required: true });
const config = defineModel<string>("config", { default: "" });

const { t } = useI18n();

/** 结构化字段 → JSON 文本；内容一致时跳过，省掉一次无谓的父子来回 */
const writeConfig = () => {
  const text = stringifyDriveConfig(driveForm.value);
  if (text !== config.value) config.value = text;
};

/** JSON 文本 → 结构化字段；文本为空或无法解析（新增、切换驱动）时反向生成默认配置 */
const readConfig = (raw?: string) => {
  const parsed = parseDriveConfigJson(raw ?? "", SIEMENS_CONFIG_DEFAULTS);
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
