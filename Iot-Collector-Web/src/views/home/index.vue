<template>
  <div class="home-container">
    <!-- 欢迎卡片 -->
    <el-card class="welcome-card" shadow="hover">
      <div class="welcome-content">
        <div class="welcome-icon">
          <el-icon :size="60"><Monitor /></el-icon>
        </div>
        <div class="welcome-text">
          <h1>{{ t("home.welcome") }}</h1>
          <p class="subtitle">{{ t("home.subtitle") }}</p>
          <p class="description">{{ t("home.description") }}</p>
        </div>
      </div>
    </el-card>

    <!-- 快捷入口 -->
    <el-row :gutter="20" class="quick-entry">
      <el-col :span="6">
        <el-card shadow="hover" class="entry-card" @click="$router.push('/point-manage/drive')">
          <div class="entry-icon drive">
            <el-icon :size="40"><Cpu /></el-icon>
          </div>
          <div class="entry-text">
            <h3>{{ t("menus.pureDrive") }}</h3>
            <p>{{ t("home.driveDesc") }}</p>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="entry-card" @click="$router.push('/point-manage/point')">
          <div class="entry-icon point">
            <el-icon :size="40"><Connection /></el-icon>
          </div>
          <div class="entry-text">
            <h3>{{ t("menus.purePoint") }}</h3>
            <p>{{ t("home.pointDesc") }}</p>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="entry-card" @click="$router.push('/point-manage/alarm')">
          <div class="entry-icon alarm">
            <el-icon :size="40"><Bell /></el-icon>
          </div>
          <div class="entry-text">
            <h3>{{ t("menus.pureAlarm") }}</h3>
            <p>{{ t("home.alarmDesc") }}</p>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover" class="entry-card" @click="$router.push('/system/info')">
          <div class="entry-icon system">
            <el-icon :size="40"><DataLine /></el-icon>
          </div>
          <div class="entry-text">
            <h3>{{ t("menus.pureSystemInfo") }}</h3>
            <p>{{ t("home.systemInfoDesc") }}</p>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 系统信息概览 -->
    <el-row :gutter="20" class="system-overview">
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <el-icon><InfoFilled /></el-icon>
              <span>{{ t("home.systemOverview") }}</span>
            </div>
          </template>
          <el-descriptions :column="1" border>
            <el-descriptions-item :label="t('home.os')">
              {{ systemInfo.OS?.Name || "-" }} {{ systemInfo.OS?.Version || "" }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('home.arch')">
              {{ systemInfo.OS?.Arch || "-" }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('home.hostname')">
              {{ systemInfo.Host?.Hostname || "-" }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('home.uptime')">
              {{ systemInfo.OS?.Uptime || "-" }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <el-icon><DataAnalysis /></el-icon>
              <span>{{ t("home.performanceOverview") }}</span>
            </div>
          </template>
          <div class="performance-items">
            <div class="perf-item">
              <span class="perf-label">{{ t("home.cpuUsage") }}</span>
              <el-progress 
                :percentage="Number(systemInfo.CPU?.UsagePercent?.toFixed(1) || 0)" 
                :color="getColor(systemInfo.CPU?.UsagePercent || 0)"
              />
            </div>
            <div class="perf-item">
              <span class="perf-label">{{ t("home.memUsage") }}</span>
              <el-progress 
                :percentage="Number(systemInfo.Memory?.UsagePercent?.toFixed(1) || 0)" 
                :color="getColor(systemInfo.Memory?.UsagePercent || 0)"
              />
            </div>
            <div class="perf-item">
              <span class="perf-label">{{ t("home.diskUsage") }}</span>
              <el-progress 
                :percentage="Number(systemInfo.Disk?.UsagePercent?.toFixed(1) || 0)" 
                :color="getColor(systemInfo.Disk?.UsagePercent || 0)"
              />
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { useI18n } from "vue-i18n";
import {
  Monitor,
  Cpu,
  Connection,
  Bell,
  DataLine,
  InfoFilled,
  DataAnalysis
} from "@element-plus/icons-vue";
import { configApi } from "@/api/config";

const { t } = useI18n();
const systemInfo = ref<any>({});
let refreshTimer: ReturnType<typeof setInterval> | null = null;

// 根据使用率返回颜色
const getColor = (percent: number) => {
  if (percent < 60) return "#67C23A";
  if (percent < 80) return "#E6A23C";
  return "#F56C6C";
};

// 加载系统信息
const loadSystemInfo = async () => {
  try {
    const res = await configApi.systemInfo.get() as any;
    if (res && res.Data) {
      systemInfo.value = res.Data;
    }
  } catch (error) {
    console.error("加载系统信息失败:", error);
  }
};

onMounted(() => {
  loadSystemInfo();
  // 3 秒自动刷新
  refreshTimer = setInterval(loadSystemInfo, 3000);
});

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
});
</script>

<style scoped lang="scss">
.home-container {
  padding: 20px;
}

.welcome-card {
  margin-bottom: 20px;

  .welcome-content {
    display: flex;
    gap: 30px;
    align-items: center;
    padding: 20px;

    .welcome-icon {
      padding: 20px;
      color: var(--ind-accent, #409eff);
      background: linear-gradient(135deg, var(--ind-bg-2, #e8f4fd) 0%, var(--ind-panel, #d4e8f7) 100%);
      border: 1px solid var(--ind-border, transparent);
      border-radius: 16px;
    }

    .welcome-text {
      h1 {
        margin: 0 0 10px;
        font-size: 28px;
        color: var(--ind-fg, var(--el-text-color-primary));
      }

      .subtitle {
        margin: 0 0 8px;
        font-size: 16px;
        color: var(--ind-fg-muted, var(--el-text-color-regular));
      }

      .description {
        margin: 0;
        font-size: 14px;
        color: var(--ind-fg-dim, var(--el-text-color-secondary));
      }
    }
  }
}

.quick-entry {
  margin-bottom: 20px;
  
  .entry-card {
    height: 100%;
    cursor: pointer;
    transition: all 0.3s;
    
    &:hover {
      transform: translateY(-5px);
    }
    
    :deep(.el-card__body) {
      display: flex;
      gap: 20px;
      align-items: center;
      padding: 20px;
    }
    
    .entry-icon {
      padding: 15px;
      color: white;
      border-radius: 12px;
      
      &.drive {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      }
      
      &.point {
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
      }
      
      &.alarm {
        background: linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%);
      }
      
      &.system {
        color: #333;
        background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%);
      }
    }
    
    .entry-text {
      h3 {
        margin: 0 0 8px;
        font-size: 16px;
        color: var(--ind-fg, var(--el-text-color-primary));
      }

      p {
        margin: 0;
        font-size: 13px;
        color: var(--ind-fg-dim, var(--el-text-color-secondary));
      }
    }
  }
}

.system-overview {
  .card-header {
    display: flex;
    gap: 8px;
    align-items: center;
    font-size: 16px;
    font-weight: 600;
    
    .el-icon {
      color: var(--ind-accent, var(--el-color-primary));
    }
  }
  
  .performance-items {
    .perf-item {
      margin-bottom: 20px;
      
      &:last-child {
        margin-bottom: 0;
      }
      
      .perf-label {
        display: block;
        margin-bottom: 8px;
        font-size: 14px;
        color: var(--ind-fg-muted, var(--el-text-color-regular));
      }
    }
  }
}
</style>
