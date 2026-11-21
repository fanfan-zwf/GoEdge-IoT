<template>
  <div class="main">
    <el-card shadow="never" class="config-card">
      <template #header>
        <div class="card-header">
          <div class="card-header-left">
            <span class="card-title">{{ t("systemConfig.title") }}</span>
            <el-tag v-if="isModified" type="warning" size="small" effect="dark">
              {{ t("systemConfig.modified") }}
            </el-tag>
            <el-tag v-else type="success" size="small" effect="dark">
              {{ t("systemConfig.saved") }}
            </el-tag>
          </div>
          <div class="header-actions">
            <el-button :disabled="isModified" @click="handleReload">
              <el-icon><Refresh /></el-icon>
              {{ t("systemConfig.reload") }}
            </el-button>
            <el-button type="primary" :loading="saving" :disabled="!isModified" @click="handleSave">
              <el-icon><Check /></el-icon>
              {{ t("systemConfig.save") }}
            </el-button>
          </div>
        </div>
      </template>

      <!-- 加载状态 -->
      <div v-if="loading" class="loading-container">
        <el-skeleton :rows="20" animated />
      </div>

      <!-- 配置编辑器 -->
      <div v-else class="editor-container">
        <!-- 工具栏 -->
        <div class="editor-toolbar">
          <div class="toolbar-left">
            <el-icon><Document /></el-icon>
            <span class="file-path">configs/config.yaml</span>
            <el-tag type="info" size="small">YAML</el-tag>
          </div>
          <div class="toolbar-right">
            <el-tooltip :content="t('systemConfig.download')" placement="top">
              <el-button :icon="Download" circle size="small" @click="handleDownload" />
            </el-tooltip>
            <el-tooltip :content="t('systemConfig.upload')" placement="top">
              <el-button :icon="Upload" circle size="small" @click="handleUploadClick" />
            </el-tooltip>
            <el-tooltip :content="t('systemConfig.reset')" placement="top">
              <el-button :icon="RefreshLeft" circle size="small" :disabled="!isModified" @click="handleReset" />
            </el-tooltip>
          </div>
        </div>

        <!-- 编辑器主体 - 带行号 -->
        <div class="editor-body">
          <div ref="lineNumbersRef" class="line-numbers">
            <div v-for="n in lineCount" :key="n" class="line-number">{{ n }}</div>
          </div>
          <el-input
            ref="editorRef"
            v-model="configContent"
            type="textarea"
            class="yaml-editor"
            spellcheck="false"
            @input="handleInput"
          />
        </div>

        <!-- 底部状态栏 -->
        <div class="editor-footer">
          <div class="footer-left">
            <span class="save-tip">{{ t('systemConfig.saveTip') }}</span>
          </div>
          <div class="footer-right">
            <span class="status-info">
              {{ t("systemConfig.lines") }}: {{ lineCount }} | {{ t("systemConfig.chars") }}: {{ configContent.length }}
            </span>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 隐藏的文件上传 input -->
    <input
      ref="fileInputRef"
      type="file"
      accept=".yaml,.yml"
      style="display: none"
      @change="handleFileChange"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  Check,
  Refresh,
  Document,
  Download,
  Upload,
  RefreshLeft
} from "@element-plus/icons-vue";
import { configApi } from "@/api/config";

const { t } = useI18n();

// 响应式数据
const loading = ref(true);
const saving = ref(false);
const configContent = ref("");
const originalContent = ref(""); // 保存原始内容用于对比
const editorRef = ref();
const lineNumbersRef = ref<HTMLElement>();
const fileInputRef = ref<HTMLInputElement>();

// 计算属性
const isModified = computed(() => configContent.value !== originalContent.value);
const lineCount = computed(() => configContent.value.split("\n").length);

// 加载配置
const loadConfig = async () => {
  loading.value = true;
  try {
    const res = await configApi.systemConfig.read() as any;
    // res 格式为 {Code, Msg, Data}，YAML 内容在 Data 字段
    const data = res.Data;
    if (data && typeof data === "string") {
      configContent.value = data;
      originalContent.value = data;
    }
  } catch (error) {
    ElMessage.error(t("systemConfig.loadFailed"));
    console.error("加载配置失败:", error);
  } finally {
    loading.value = false;
    // 加载完成后绑定滚动事件
    nextTick(() => bindScrollEvent());
  }
};

// 处理输入
const handleInput = () => {
  // 输入时自动标记为已修改（computed 会自动处理）
};

// 编辑器滚动时同步行号
const handleEditorScroll = () => {
  const textarea = editorRef.value?.textarea as HTMLElement | undefined;
  if (textarea && lineNumbersRef.value) {
    lineNumbersRef.value.scrollTop = textarea.scrollTop;
  }
};

// 绑定 textarea 滚动事件
let boundTextarea: HTMLElement | null = null;
const bindScrollEvent = () => {
  const textarea = editorRef.value?.textarea as HTMLElement | undefined;
  if (textarea && textarea !== boundTextarea) {
    // 移除旧的监听
    if (boundTextarea) {
      boundTextarea.removeEventListener("scroll", handleEditorScroll);
    }
    textarea.addEventListener("scroll", handleEditorScroll);
    boundTextarea = textarea;
  }
};

// 重新加载配置
const handleReload = async () => {
  try {
    await ElMessageBox.confirm(
      t("systemConfig.reloadConfirm"),
      t("common.warning"),
      {
        confirmButtonText: t("common.confirm"),
        cancelButtonText: t("common.cancel"),
        type: "warning"
      }
    );

    await configApi.systemConfig.reload();
    ElMessage.success(t("systemConfig.reloadSuccess"));
    await loadConfig();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error(t("systemConfig.reloadFailed"));
      console.error("重新加载失败:", error);
    }
  }
};

// 保存配置
const handleSave = async () => {
  try {
    if (!configContent.value.trim()) {
      ElMessage.warning(t("systemConfig.contentEmpty"));
      return;
    }

    saving.value = true;
    await configApi.systemConfig.write({ content: configContent.value });
    originalContent.value = configContent.value;
    ElMessage.success(t("systemConfig.saveSuccess"));
  } catch (error) {
    ElMessage.error(t("systemConfig.saveFailed") + ": " + (error as Error).message);
    console.error("保存失败:", error);
  } finally {
    saving.value = false;
  }
};

// 下载配置文件
const handleDownload = () => {
  const blob = new Blob([configContent.value], { type: "text/yaml" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "config.yaml";
  a.click();
  URL.revokeObjectURL(url);
  ElMessage.success(t("systemConfig.downloadSuccess"));
};

// 点击上传按钮
const handleUploadClick = () => {
  fileInputRef.value?.click();
};

// 处理文件选择
const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  const reader = new FileReader();
  reader.onload = e => {
    const content = e.target?.result as string;
    configContent.value = content;
    ElMessage.success(t("systemConfig.uploadSuccess"));
  };
  reader.readAsText(file);

  // 重置 input 以便再次选择同一文件
  target.value = "";
};

// 重置为原始内容
const handleReset = async () => {
  try {
    await ElMessageBox.confirm(
      t("systemConfig.resetConfirm"),
      t("common.warning"),
      {
        confirmButtonText: t("common.confirm"),
        cancelButtonText: t("common.cancel"),
        type: "warning"
      }
    );
    configContent.value = originalContent.value;
    ElMessage.info(t("systemConfig.resetSuccess"));
  } catch {
    // 用户取消
  }
};

// 键盘快捷键
const handleKeydown = (e: KeyboardEvent) => {
  // Ctrl+S / Cmd+S 保存
  if ((e.ctrlKey || e.metaKey) && e.key === "s") {
    e.preventDefault();
    if (isModified.value) {
      handleSave();
    }
  }
};

// 初始化
onMounted(() => {
  loadConfig();
  document.addEventListener("keydown", handleKeydown);
});

// 清理
onUnmounted(() => {
  document.removeEventListener("keydown", handleKeydown);
  if (boundTextarea) {
    boundTextarea.removeEventListener("scroll", handleEditorScroll);
    boundTextarea = null;
  }
});
</script>

<style scoped lang="scss">
.main {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 130px);
  padding: 0;
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;

  .card-header-left {
    display: flex;
    gap: 12px;
    align-items: center;

    .card-title {
      font-size: 16px;
      font-weight: 600;
    }
  }

  .header-actions {
    display: flex;
    gap: 10px;
  }
}

.loading-container {
  padding: 20px 0;
}

:deep(.config-card) {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  border: none;

  .el-card__header {
    flex-shrink: 0;
    padding: 12px 20px;
  }

  .el-card__body {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    padding: 0 20px 16px;
    overflow: hidden;
  }
}

.editor-container {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}

// 工具栏
.editor-toolbar {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background-color: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color);

  .toolbar-left {
    display: flex;
    gap: 8px;
    align-items: center;
    color: var(--el-text-color-regular);

    .file-path {
      font-family: "Fira Code", Consolas, monospace;
      font-size: 13px;
    }
  }

  .toolbar-right {
    display: flex;
    gap: 8px;
  }
}

// 编辑器主体
.editor-body {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  background-color: #1e1e1e;

  // 行号
  .line-numbers {
    flex-shrink: 0;
    min-width: 50px;
    padding: 12px 0;
    overflow: hidden;
    text-align: right;
    user-select: none;
    background-color: #252526;
    border-right: 1px solid #3e3e42;

    .line-number {
      height: 22.4px;
      padding: 0 12px 0 8px;
      font-family: "Fira Code", Consolas, monospace;
      font-size: 14px;
      line-height: 1.6;
      color: #858585;
    }
  }

  // 编辑器
  .yaml-editor {
    flex: 1;
    overflow: hidden;

    :deep(.el-textarea) {
      height: 100%;
    }

    :deep(.el-textarea__inner) {
      height: 100% !important;
      padding: 12px 16px;
      overflow-y: auto !important;
      font-family: "Fira Code", Consolas, Monaco, monospace;
      font-size: 14px;
      line-height: 1.6;
      color: #d4d4d4;
      tab-size: 2;
      resize: none !important;
      background-color: #1e1e1e;
      border: none;
      box-shadow: none;

      &:focus {
        box-shadow: none !important;
      }
    }
  }
}

// 底部状态栏
.editor-footer {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  background-color: var(--el-fill-color-lighter);
  border-top: 1px solid var(--el-border-color);

  .footer-left {
    flex: 1;

    .save-tip {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .footer-right {
    .status-info {
      font-family: "Fira Code", Consolas, monospace;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }
}
</style>
