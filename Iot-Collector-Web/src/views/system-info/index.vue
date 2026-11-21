<template>
  <div class="main">
    <!-- 系统概览卡片 -->
    <el-row :gutter="20" class="info-row">
      <el-col :span="6">
        <el-card shadow="hover" class="info-card os-card">
          <div class="card-icon">
            <el-icon :size="40"><Monitor /></el-icon>
          </div>
          <div class="card-content">
            <div class="card-title">{{ t("systemInfo.os") }}</div>
            <div class="card-value">{{ systemInfo.OS?.Name || "-" }}</div>
            <div class="card-desc">{{ systemInfo.OS?.Version || "-" }}</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="info-card arch-card">
          <div class="card-icon">
            <el-icon :size="40"><Cpu /></el-icon>
          </div>
          <div class="card-content">
            <div class="card-title">{{ t("systemInfo.arch") }}</div>
            <div class="card-value">{{ systemInfo.OS?.Arch || "-" }}</div>
            <div class="card-desc">
              <img
                v-if="osLogo"
                :src="osLogo"
                :alt="systemInfo.OS?.Platform"
                class="os-logo"
              />
              {{ systemInfo.OS?.Platform || "-" }}
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="info-card uptime-card">
          <div class="card-icon">
            <el-icon :size="40"><Timer /></el-icon>
          </div>
          <div class="card-content">
            <div class="card-title">{{ t("systemInfo.uptime") }}</div>
            <div class="card-value">{{ systemInfo.OS?.Uptime || "-" }}</div>
            <div class="card-desc">{{ systemInfo.Host?.Hostname || "-" }}</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="info-card time-card">
          <div class="card-icon">
            <el-icon :size="40"><Clock /></el-icon>
          </div>
          <div class="card-content">
            <div class="card-title">{{ t("systemInfo.currentTime") }}</div>
            <div class="card-value">{{ currentTime }}</div>
            <div class="card-desc">{{ t("systemInfo.lastUpdate") }}: {{ systemInfo.Timestamp || "-" }}</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 详细性能信息 -->
    <el-row :gutter="20" class="detail-row">
      <!-- CPU 信息 -->
      <el-col :span="12">
        <el-card shadow="never" class="detail-card">
          <template #header>
            <div class="detail-header">
              <el-icon><Cpu /></el-icon>
              <span>{{ t("systemInfo.cpuInfo") }}</span>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item :label="t('systemInfo.cpuModel')">
              {{ systemInfo.CPU?.ModelName || "-" }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('systemInfo.cpuCores')">
              {{ systemInfo.CPU?.PhysicalCores || "-" }} {{ t("systemInfo.physicalCores") }} / 
              {{ systemInfo.CPU?.Cores || "-" }} {{ t("systemInfo.logicalCores") }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('systemInfo.cpuFrequency')">
              {{ systemInfo.CPU?.Frequency?.toFixed(0) || "-" }} MHz
            </el-descriptions-item>
            <el-descriptions-item :label="t('systemInfo.cpuUsage')">
              <div class="progress-container">
                <el-progress 
                  :percentage="Number(systemInfo.CPU?.UsagePercent?.toFixed(1) || 0)" 
                  :color="getCpuColor"
                  :stroke-width="20"
                  :text-inside="true"
                />
              </div>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <!-- 内存信息 -->
      <el-col :span="12">
        <el-card shadow="never" class="detail-card">
          <template #header>
            <div class="detail-header">
              <el-icon><Coin /></el-icon>
              <span>{{ t("systemInfo.memoryInfo") }}</span>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item :label="t('systemInfo.memTotal')">
              {{ formatBytes(systemInfo.Memory?.Total) }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('systemInfo.memUsed')">
              {{ formatBytes(systemInfo.Memory?.Used) }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('systemInfo.memAvailable')">
              {{ formatBytes(systemInfo.Memory?.Available) }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('systemInfo.memUsage')">
              <div class="progress-container">
                <el-progress 
                  :percentage="Number(systemInfo.Memory?.UsagePercent?.toFixed(1) || 0)" 
                  :color="getMemoryColor"
                  :stroke-width="20"
                  :text-inside="true"
                />
              </div>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="detail-row">
      <!-- 交换内存 -->
      <el-col :span="12">
        <el-card shadow="never" class="detail-card">
          <template #header>
            <div class="detail-header">
              <el-icon><Files /></el-icon>
              <span>{{ t("systemInfo.swapInfo") }}</span>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item :label="t('systemInfo.swapTotal')">
              {{ formatBytes(systemInfo.Memory?.SwapTotal) }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('systemInfo.swapUsed')">
              {{ formatBytes(systemInfo.Memory?.SwapUsed) }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('systemInfo.swapUsage')">
              <div class="progress-container">
                <el-progress 
                  :percentage="Number(systemInfo.Memory?.SwapUsagePercent?.toFixed(1) || 0)" 
                  :color="getSwapColor"
                  :stroke-width="20"
                  :text-inside="true"
                />
              </div>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <!-- 硬盘信息 -->
      <el-col :span="12">
        <el-card shadow="never" class="detail-card">
          <template #header>
            <div class="detail-header">
              <el-icon><Box /></el-icon>
              <span>{{ t("systemInfo.diskInfo") }}</span>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item :label="t('systemInfo.diskTotal')">
              {{ formatBytes(systemInfo.Disk?.Total) }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('systemInfo.diskUsed')">
              {{ formatBytes(systemInfo.Disk?.Used) }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('systemInfo.diskFree')">
              {{ formatBytes(systemInfo.Disk?.Free) }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('systemInfo.diskUsage')">
              <div class="progress-container">
                <el-progress 
                  :percentage="Number(systemInfo.Disk?.UsagePercent?.toFixed(1) || 0)" 
                  :color="getDiskColor"
                  :stroke-width="20"
                  :text-inside="true"
                />
              </div>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <!-- 刷新控制 -->
    <div class="refresh-container">
      <el-button type="primary" :loading="loading" @click="loadSystemInfo">
        <el-icon><Refresh /></el-icon>
        {{ t("systemInfo.refresh") }}
      </el-button>
      <el-switch
        v-model="autoRefresh"
        :active-text="t('systemInfo.autoRefresh')"
        @change="toggleAutoRefresh"
      />
      <el-select
        v-model="refreshInterval"
        :disabled="!autoRefresh"
        style="width: 120px"
        @change="handleIntervalChange"
      >
        <el-option :label="'1 ' + t('systemInfo.second')" :value="1000" />
        <el-option :label="'5 ' + t('systemInfo.seconds')" :value="5000" />
        <el-option :label="'10 ' + t('systemInfo.seconds')" :value="10000" />
      </el-select>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from "vue";
import { useI18n } from "vue-i18n";
import { ElMessage } from "element-plus";
import {
  Monitor,
  Cpu,
  Timer,
  Clock,
  Coin,
  Files,
  Box,
  Refresh
} from "@element-plus/icons-vue";
import { configApi } from "@/api/config";

const { t } = useI18n();

// 响应式数据
const loading = ref(false);
const autoRefresh = ref(true); // 默认开启自动刷新
const refreshInterval = ref(5000); // 默认 5 秒
const systemInfo = ref<any>({});
const currentTime = ref("");
let refreshTimer: ReturnType<typeof setInterval> | null = null;
let clockTimer: ReturnType<typeof setInterval> | null = null;

// 系统 Logo 计算属性
const osLogo = computed(() => {
  const platform = systemInfo.value?.OS?.Platform?.toLowerCase() || "";
  // 使用简单的 SVG 图标或者 emoji
  const logoMap: Record<string, string> = {
    ubuntu: "https://cdn.simpleicons.org/ubuntu/E95420",
    debian: "https://cdn.simpleicons.org/debian/A81D33",
    centos: "https://cdn.simpleicons.org/centos/262577",
    rhel: "https://cdn.simpleicons.org/redhat/EE0000",
    fedora: "https://cdn.simpleicons.org/fedora/51A2DA",
    arch: "https://cdn.simpleicons.org/archlinux/1793D1",
    manjaro: "https://cdn.simpleicons.org/manjaro/35BF5C",
    linux: "https://cdn.simpleicons.org/linux/FCC624",
    windows: "https://cdn.simpleicons.org/windows/0078D6",
    "microsoft": "https://cdn.simpleicons.org/windows/0078D6",
    darwin: "https://cdn.simpleicons.org/apple/000000",
    macos: "https://cdn.simpleicons.org/apple/000000",
    apple: "https://cdn.simpleicons.org/apple/000000"
  };
  
  // 查找匹配的 logo
  for (const [key, url] of Object.entries(logoMap)) {
    if (platform.includes(key)) {
      return url;
    }
  }
  return "";
});

// 颜色配置
const getCpuColor = computed(() => [
  { color: "#67C23A", percentage: 50 },
  { color: "#E6A23C", percentage: 80 },
  { color: "#F56C6C", percentage: 100 }
]);

const getMemoryColor = computed(() => [
  { color: "#67C23A", percentage: 60 },
  { color: "#E6A23C", percentage: 80 },
  { color: "#F56C6C", percentage: 100 }
]);

const getSwapColor = computed(() => [
  { color: "#67C23A", percentage: 50 },
  { color: "#E6A23C", percentage: 80 },
  { color: "#F56C6C", percentage: 100 }
]);

const getDiskColor = computed(() => [
  { color: "#67C23A", percentage: 70 },
  { color: "#E6A23C", percentage: 85 },
  { color: "#F56C6C", percentage: 100 }
]);

// 格式化字节
const formatBytes = (bytes: number | undefined): string => {
  if (!bytes || bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const k = 1024;
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return (bytes / Math.pow(k, i)).toFixed(2) + " " + units[i];
};

// 加载系统信息
const loadSystemInfo = async () => {
  loading.value = true;
  try {
    const res = await configApi.systemInfo.get() as any;
    if (res && res.Data) {
      systemInfo.value = res.Data;
    }
  } catch (error) {
    ElMessage.error(t("systemInfo.loadFailed"));
    console.error("加载系统信息失败:", error);
  } finally {
    loading.value = false;
  }
};

// 更新时间
const updateTime = () => {
  const now = new Date();
  currentTime.value = now.toLocaleString("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false
  });
};

// 切换自动刷新
const toggleAutoRefresh = (val: boolean) => {
  if (val) {
    refreshTimer = setInterval(loadSystemInfo, refreshInterval.value);
  } else {
    if (refreshTimer) {
      clearInterval(refreshTimer);
      refreshTimer = null;
    }
  }
};

// 切换刷新间隔
const handleIntervalChange = () => {
  // 如果自动刷新已开启，重新启动定时器
  if (autoRefresh.value) {
    if (refreshTimer) {
      clearInterval(refreshTimer);
    }
    refreshTimer = setInterval(loadSystemInfo, refreshInterval.value);
  }
};

// 初始化
onMounted(() => {
  loadSystemInfo();
  updateTime();
  clockTimer = setInterval(updateTime, 1000);
  // 启动自动刷新
  if (autoRefresh.value) {
    refreshTimer = setInterval(loadSystemInfo, refreshInterval.value);
  }
});

// 清理
onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer);
  if (clockTimer) clearInterval(clockTimer);
});
</script>

<style scoped lang="scss">
.main {
  padding: 20px;
}

.info-row {
  margin-bottom: 20px;
}

.info-card {
  display: flex;
  align-items: center;
  padding: 20px;
  
  :deep(.el-card__body) {
    display: flex;
    align-items: center;
    width: 100%;
    padding: 0;
  }

  .card-icon {
    padding: 15px;
    margin-right: 20px;
    color: white;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    border-radius: 12px;
    
    .os-card & {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    }
    
    .arch-card & {
      background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
    }
    
    .uptime-card & {
      background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
    }
    
    .time-card & {
      background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
    }
  }

  .card-content {
    flex: 1;

    .card-title {
      margin-bottom: 6px;
      font-size: 13px;
      color: var(--ind-fg-muted, var(--el-text-color-secondary));
      letter-spacing: 0.5px;
    }

    .card-value {
      margin-bottom: 4px;
      font-size: 22px;
      font-weight: 700;
      color: var(--ind-fg, var(--el-text-color-primary));
      text-shadow: 0 0 12px var(--ind-accent-glow, transparent);
    }

    .card-desc {
      font-size: 12px;
      color: var(--ind-fg-muted, var(--el-text-color-secondary));
    }
  }
}

.detail-row {
  margin-bottom: 20px;
}

.detail-card {
  height: 100%;

  :deep(.el-card__header) {
    color: var(--ind-fg, var(--el-text-color-primary));
  }

  .detail-header {
    display: flex;
    gap: 8px;
    align-items: center;
    font-size: 16px;
    font-weight: 600;
    color: var(--ind-fg, var(--el-text-color-primary));

    .el-icon {
      color: var(--ind-accent, var(--el-color-primary));
    }
  }

  :deep(.el-descriptions) {
    color: var(--ind-fg, var(--el-text-color-primary));
  }

  :deep(.el-descriptions__label) {
    color: var(--ind-fg-muted, var(--el-text-color-secondary)) !important;
  }

  :deep(.el-descriptions__content) {
    color: var(--ind-fg, var(--el-text-color-primary)) !important;
  }
}

.progress-container {
  width: 100%;
  padding: 5px 0;
}

.os-logo {
  width: 16px;
  height: 16px;
  margin-right: 4px;
  vertical-align: middle;
}

.refresh-container {
  display: flex;
  gap: 20px;
  align-items: center;
  justify-content: center;
  padding: 20px;
  margin-top: 20px;
  color: var(--ind-fg-muted, var(--el-text-color-regular));
  background: var(--ind-panel-2, var(--el-fill-color-lighter));
  border: 1px solid var(--ind-border, var(--el-border-color-lighter));
  border-radius: 8px;
}
</style>
