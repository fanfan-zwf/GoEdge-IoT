<script setup lang="ts">
import Motion from "../utils/motion";
import TypeIt from "@/components/ReTypeit";
import { $t, transformI18n } from "@/plugins/i18n";
import { languageOptions } from "@/layout/hooks/useTranslationLang";
import { useRenderIcon } from "@/components/ReIcon/src/hooks";
import { loginRules } from "../utils/rule";
import { useLogin } from "../composables/useLogin";
import { debounce } from "@pureadmin/utils";
import { ref, watch, onMounted } from "vue";
import { message } from "@/utils/message";
import { getBaseServer, setBaseServer } from "@/api/config";
import {
  normalizeServerAddr,
  getServerHistory,
  addServerHistory,
  removeServerHistory,
  clearServerHistory
} from "@/utils/serverAddr";

import globalization from "@/assets/svg/globalization.svg?component";
import Lock from "~icons/ri/lock-fill";
import Check from "~icons/ep/check";
import User from "~icons/ri/user-3-fill";
import Server from "~icons/ri/server-line";
import PlcSilhouette from "./svg/plc-silhouette.svg?component";

defineOptions({ name: "LoginVariantA" });

const {
  t,
  title,
  ruleForm,
  ruleFormRef,
  loading,
  disabled,
  onLogin,
  locale,
  currentLangLabel,
  translationLang,
  getDropdownItemStyle,
  getDropdownItemClass
} = useLogin();

// ===== 高级设置：服务器地址（持久化，刷新后自动恢复） =====
const advancedOpen = ref(false);
const serverHistory = ref<string[]>([]);
// 输入框初始留空，避免拿预填的完整地址去过滤建议（否则聚焦只显示自己一条）；
// 当前生效地址靠 appliedServer / placeholder 展示，不靠输入框预填。
const serverAddr = ref("");
const appliedServer = ref(getBaseServer());

// applyLive：将输入框地址写入内存覆盖（不持久化、不记历史），供 HTTP/WS 实时取用
const applyLive = () => {
  const val = normalizeServerAddr(serverAddr.value);
  if (!val) return;
  setBaseServer(val, false);
  appliedServer.value = val;
};
const applyLiveDebounced = debounce(applyLive, 400);

// commit：应用并持久化当前地址 + 记入历史（选中建议 / 点“保存并使用”时触发）
const commit = (raw?: string) => {
  if (typeof raw === "string") serverAddr.value = raw;
  const val = normalizeServerAddr(serverAddr.value);
  if (!val) return;
  setBaseServer(val, true);
  appliedServer.value = val;
  serverHistory.value = addServerHistory(val);
  message(t("login.serverSwitched", { server: val }), { type: "success" });
};

const toggleAdvanced = () => {
  advancedOpen.value = !advancedOpen.value;
  if (advancedOpen.value) serverHistory.value = getServerHistory();
};

const fetchSuggestions = (queryString: string, cb: (s: { value: string }[]) => void) => {
  const list = getServerHistory();
  const q = (queryString || "").trim().toLowerCase();
  const matched = q ? list.filter(v => v.toLowerCase().includes(q)) : list;
  cb(matched.map(v => ({ value: v })));
};

const removeRecord = (v: string) => {
  serverHistory.value = removeServerHistory(v);
};
const clearRecords = () => {
  serverHistory.value = clearServerHistory();
};

// 输入时防抖实时生效，保证 Enter 登录路径也能使用手输地址
watch(serverAddr, () => applyLiveDebounced());

onMounted(() => {
  serverHistory.value = getServerHistory();
});

// 登录前先固化当前地址（持久化覆盖 + 记历史），再走原有登录逻辑
const loginWithServer = () => {
  const val = normalizeServerAddr(serverAddr.value);
  if (val) {
    setBaseServer(val, true);
    appliedServer.value = val;
    serverHistory.value = addServerHistory(val);
  }
  onLogin(ruleFormRef.value);
};
</script>

<template>
  <div class="va-root select-none">
    <!-- 背景层：渐变 + 等距工程网格 -->
    <div class="va-bg" aria-hidden="true" />
    <div class="va-grid" aria-hidden="true" />

    <!-- 远景 PLC 机柜剪影 -->
    <div class="va-cabinet" aria-hidden="true">
      <PlcSilhouette />
    </div>

    <!-- 横向信号带：从左到右流动 packets -->
    <div class="va-signal-band" aria-hidden="true">
      <div class="va-packet va-packet-1" />
      <div class="va-packet va-packet-2" />
      <div class="va-packet va-packet-3" />
      <div class="va-packet va-packet-4" />
      <div class="va-packet va-packet-5" />
      <div class="va-packet va-packet-6" />
    </div>

    <!-- 顶部工具栏（仅保留语言切换，浅色/深色开关已移除） -->
    <div class="va-toolbar">
      <el-dropdown trigger="click">
        <div class="va-lang flex cursor-pointer items-center gap-1 outline-hidden">
          <globalization class="size-5" />
          <span class="text-sm whitespace-nowrap">{{ currentLangLabel }}</span>
        </div>
        <template #dropdown>
          <el-dropdown-menu class="translation">
            <el-dropdown-item
              v-for="lang in languageOptions"
              :key="lang.locale"
              :style="getDropdownItemStyle(locale, lang.locale)"
              :class="['dark:text-white!', getDropdownItemClass(locale, lang.locale)]"
              @click="translationLang(lang.locale)"
            >
              <IconifyIconOffline
                v-show="locale === lang.locale"
                :icon="Check"
                style="position: absolute; left: 20px"
              />
              {{ lang.label }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>

    <!-- 主布局 -->
    <div class="va-main">
      <div class="va-brand">
        <div class="va-brand-eyebrow">INDUSTRIAL DATA GATEWAY</div>
        <h1 class="va-brand-title">
          <TypeIt :options="{ strings: [title], cursor: false, speed: 100 }" />
        </h1>
        <p class="va-brand-sub">
          {{ t("login.brandSub") }}
        </p>
        <div class="va-brand-badges">
          <span class="va-badge">MODBUS TCP</span>
          <span class="va-badge">SIEMENS S7</span>
          <span class="va-badge">MQTT</span>
          <span class="va-badge">INFLUXDB</span>
        </div>
      </div>

      <!-- 登录卡片 -->
      <div class="va-card">
        <div class="va-card-header">
          <span class="va-led led-green"><i /></span>
          <span class="va-led-text">ONLINE</span>
          <span class="va-led led-amber"><i /></span>
          <span class="va-led-text">SYNC</span>
          <span class="va-led led-red"><i /></span>
          <span class="va-led-text">ALARM</span>
          <span class="va-card-tag">// SECURE TERMINAL</span>
        </div>

        <div class="va-card-body">
          <h2 class="va-title outline-hidden">{{ t("login.systemLogin") }}</h2>
          <p class="va-subtitle">SIGN IN TO CONSOLE</p>

          <el-form ref="ruleFormRef" :model="ruleForm" :rules="loginRules" size="large">
            <Motion :delay="100">
              <el-form-item
                :rules="[
                  {
                    required: true,
                    message: transformI18n($t('login.pureUsernameReg')),
                    trigger: 'blur'
                  }
                ]"
                prop="username"
              >
                <el-input
                  v-model="ruleForm.username"
                  clearable
                  :placeholder="t('login.pureUsername')"
                  :prefix-icon="useRenderIcon(User)"
                />
              </el-form-item>
            </Motion>

            <Motion :delay="150">
              <el-form-item prop="password">
                <el-input
                  v-model="ruleForm.password"
                  clearable
                  show-password
                  :placeholder="t('login.purePassword')"
                  :prefix-icon="useRenderIcon(Lock)"
                />
              </el-form-item>
            </Motion>

            <Motion :delay="200">
              <el-button
                class="va-btn"
                type="primary"
                :loading="loading"
                :disabled="disabled"
                @click="loginWithServer"
              >
                {{ t("login.pureLogin") }}
              </el-button>
            </Motion>
          </el-form>

          <!-- 高级设置：服务器地址（临时覆盖，仅本次会话生效） -->
          <div class="va-advanced">
            <div class="va-advanced-toggle" @click="toggleAdvanced">
              <span class="va-caret" :class="{ 'va-caret-open': advancedOpen }">▸</span>
              <span>{{ t("login.advancedSettings") }}</span>
            </div>
            <el-collapse-transition>
              <div v-show="advancedOpen" class="va-advanced-body">
                <el-autocomplete
                  v-model="serverAddr"
                  :fetch-suggestions="fetchSuggestions"
                  :prefix-icon="useRenderIcon(Server)"
                  :placeholder="serverHistory.length ? t('login.serverPlaceholderHasHistory', { server: appliedServer }) : t('login.serverPlaceholderEmpty')"
                  clearable
                  class="va-server-input"
                  @select="(item: any) => commit(item.value)"
                >
                  <template #default="{ item }">
                    <div class="va-sug flex w-full items-center justify-between gap-3">
                      <span class="truncate">{{ item.value }}</span>
                      <span class="va-sug-del shrink-0" @click.stop="removeRecord(item.value)">{{ t("login.deleteRecord") }}</span>
                    </div>
                  </template>
                </el-autocomplete>
                <div class="va-advanced-actions">
                  <el-button text size="small" class="va-link" @click="commit()">{{ t("login.saveAndUse") }}</el-button>
                  <el-button
                    text
                    size="small"
                    class="va-link"
                    :disabled="!serverHistory.length"
                    @click="clearRecords"
                  >
                    {{ t("login.clearHistory") }}
                  </el-button>
                </div>
                <p class="va-advanced-tip">{{ t("login.currentEffective", { server: appliedServer }) }}</p>
              </div>
            </el-collapse-transition>
          </div>
        </div>

        <div class="va-card-footer">
          <span class="va-foot-item">CH:01 · BAUD:115200 · PARITY:N</span>
          <span class="va-foot-item">v1.0 · EDGE-GW</span>
        </div>
      </div>
    </div>

    <div class="va-footer">
      Copyright © 2026
      <a
        class="hover:text-[#4dd0e1]!"
        href="https://github.com/fanfan-zwf/GoEdge-IoT"
        target="_blank"
      >
        &nbsp;GoEdge-IoT
      </a>
    </div>
  </div>
</template>

<style scoped>
.va-root {
  position: relative;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  font-family: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;
  color: #e0ecf5;
}

/* 背景层：深蓝黑渐变 */
.va-bg {
  position: absolute;
  inset: 0;
  z-index: 0;
  background:
    radial-gradient(1200px 600px at 20% 20%, rgb(41 121 255 / 18%), transparent 60%),
    radial-gradient(1000px 500px at 90% 80%, rgb(77 208 225 / 12%), transparent 55%),
    linear-gradient(135deg, #050b1e 0%, #0b1a3a 60%, #061127 100%);
}

/* 等距网格：细线 40px 单元 */
.va-grid {
  position: absolute;
  inset: 0;
  z-index: 1;
  pointer-events: none;
  background-image:
    linear-gradient(rgb(77 208 225 / 6%) 1px, transparent 1px),
    linear-gradient(90deg, rgb(77 208 225 / 6%) 1px, transparent 1px);
  background-size: 40px 40px;
  mask-image: radial-gradient(ellipse at center, black 40%, transparent 85%);
}

/* PLC 机柜剪影 */
.va-cabinet {
  position: absolute;
  bottom: 8%;
  left: 4%;
  z-index: 2;
  width: 620px;
  max-width: 45vw;
  color: #1e3a6e;
  pointer-events: none;
  opacity: 0.55;
  filter: drop-shadow(0 0 30px rgb(77 208 225 / 15%));
}

.va-cabinet :deep(svg) {
  width: 100%;
  height: auto;
}

/* 信号带 + 光点 */
.va-signal-band {
  position: absolute;
  top: 50%;
  right: 0;
  left: 0;
  z-index: 2;
  height: 2px;
  background: linear-gradient(90deg, transparent, rgb(77 208 225 / 25%), transparent);
}

.va-packet {
  position: absolute;
  top: -3px;
  width: 8px;
  height: 8px;
  background: #4dd0e1;
  border-radius: 50%;
  box-shadow:
    0 0 12px #4dd0e1,
    -14px 0 8px -4px rgb(77 208 225 / 60%),
    -28px 0 6px -6px rgb(77 208 225 / 40%);
  animation: va-flow 6s linear infinite;
}

.va-packet-2 {
  background: #3ddc97;
  box-shadow:
    0 0 12px #3ddc97,
    -14px 0 8px -4px rgb(61 220 151 / 60%),
    -28px 0 6px -6px rgb(61 220 151 / 40%);
  animation-delay: -1s;
}

.va-packet-3 {
  animation-delay: -2s;
}

.va-packet-4 {
  background: #ff8a3d;
  box-shadow:
    0 0 12px #ff8a3d,
    -14px 0 8px -4px rgb(255 138 61 / 60%),
    -28px 0 6px -6px rgb(255 138 61 / 40%);
  animation-delay: -3s;
}

.va-packet-5 {
  animation-delay: -4s;
}

.va-packet-6 {
  background: #3ddc97;
  box-shadow:
    0 0 12px #3ddc97,
    -14px 0 8px -4px rgb(61 220 151 / 60%),
    -28px 0 6px -6px rgb(61 220 151 / 40%);
  animation-delay: -5s;
}

@keyframes va-flow {
  0% {
    left: -6%;
    opacity: 0;
  }

  8% {
    opacity: 1;
  }

  92% {
    opacity: 1;
  }

  100% {
    left: 106%;
    opacity: 0;
  }
}

/* 顶部工具栏 */
.va-toolbar {
  position: absolute;
  top: 16px;
  right: 22px;
  z-index: 10;
  display: flex;
  gap: 10px;
  align-items: center;
}

.va-lang {
  color: #cfe6f2;
}

.va-lang:hover {
  color: #4dd0e1;
}

/* 主布局 */
.va-main {
  position: relative;
  z-index: 5;
  display: grid;
  grid-template-columns: 1.2fr 1fr;
  gap: 40px;
  align-items: center;
  height: 100%;
  padding: 0 6vw;
}

/* 左侧品牌 */
.va-brand {
  max-width: 560px;
}

.va-brand-eyebrow {
  margin-bottom: 12px;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 12px;
  color: #4dd0e1;
  letter-spacing: 3px;
  opacity: 0.85;
}

.va-brand-title {
  min-height: 44px;
  margin: 0 0 14px;
  font-size: 34px;
  font-weight: 700;
  line-height: 1.2;
  color: transparent;
  background: linear-gradient(90deg, #e0ecf5, #4dd0e1);
  background-clip: text;
  background-clip: text;
}

.va-brand-sub {
  margin: 0 0 22px;
  font-size: 14px;
  color: #a8c0d6;
}

.va-brand-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.va-badge {
  padding: 4px 10px;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 11px;
  color: #4dd0e1;
  letter-spacing: 1px;
  background: rgb(77 208 225 / 6%);
  border: 1px solid rgb(77 208 225 / 35%);
  border-radius: 3px;
}

/* 登录卡片 */
.va-card {
  position: relative;
  justify-self: end;
  width: 100%;
  max-width: 420px;
  padding: 0;
  overflow: hidden;
  background: rgb(12 24 48 / 55%);
  border: 1px solid rgb(77 208 225 / 35%);
  border-radius: 12px;
  box-shadow:
    0 20px 60px rgb(0 0 0 / 35%),
    inset 0 0 40px rgb(77 208 225 / 5%);
  backdrop-filter: blur(14px);
  backdrop-filter: blur(14px);
}

.va-card::before {
  position: absolute;
  inset: 0;
  padding: 1px;
  pointer-events: none;
  content: "";
  background: linear-gradient(135deg, rgb(77 208 225 / 50%), transparent 40%, rgb(41 121 255 / 40%));
  border-radius: 12px;
  mask:
    linear-gradient(#000 0 0) content-box,
    linear-gradient(#000 0 0);
  mask-composite: xor;
  mask-composite: exclude;
}

.va-card-header {
  display: flex;
  gap: 6px;
  align-items: center;
  padding: 10px 18px;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 11px;
  color: #a8c0d6;
  background: rgb(77 208 225 / 6%);
  border-bottom: 1px solid rgb(77 208 225 / 20%);
}

.va-led {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 8px;
  height: 8px;
  margin-left: 4px;
  border-radius: 50%;
}

.va-led i {
  display: block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.led-green i {
  background: #3ddc97;
  box-shadow: 0 0 8px #3ddc97;
  animation: va-pulse 1.8s ease-in-out infinite;
}

.led-amber i {
  background: #ffb547;
  box-shadow: 0 0 8px #ffb547;
  animation: va-pulse 2.2s ease-in-out infinite;
}

.led-red i {
  background: #ef5350;
  box-shadow: 0 0 6px rgb(239 83 80 / 60%);
  opacity: 0.55;
}

@keyframes va-pulse {
  0%,
  100% {
    opacity: 1;
    transform: scale(1);
  }

  50% {
    opacity: 0.4;
    transform: scale(0.85);
  }
}

.va-led-text {
  margin-right: 6px;
  letter-spacing: 1.2px;
}

.va-card-tag {
  margin-left: auto;
  color: #4dd0e1;
  opacity: 0.75;
}

.va-card-body {
  padding: 28px 32px 22px;
}

.va-title {
  margin: 0 0 4px;
  font-size: 22px;
  font-weight: 600;
  color: #e0ecf5;
  letter-spacing: 1px;
}

.va-subtitle {
  margin: 0 0 22px;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 11px;
  color: #4dd0e1;
  letter-spacing: 2px;
  opacity: 0.65;
}

.va-card-footer {
  display: flex;
  justify-content: space-between;
  padding: 10px 20px;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 10px;
  color: #6a8cae;
  letter-spacing: 0.5px;
  background: rgb(0 0 0 / 20%);
  border-top: 1px solid rgb(77 208 225 / 15%);
}

/* Element Plus 输入框主题 */
.va-card :deep(.el-input__wrapper) {
  background: rgb(6 15 30 / 60%);
  border-radius: 4px;
  box-shadow: 0 0 0 1px rgb(77 208 225 / 25%) inset;
  transition: all 0.2s;
}

.va-card :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px rgb(77 208 225 / 55%) inset;
}

.va-card :deep(.el-input__wrapper.is-focus) {
  box-shadow:
    0 0 0 1px #4dd0e1 inset,
    0 0 12px rgb(77 208 225 / 35%);
}

.va-card :deep(.el-input__inner) {
  color: #e0ecf5;
}

.va-card :deep(.el-input__prefix-inner > :first-child) {
  color: #4dd0e1;
  opacity: 0.7;
}

/* 登录按钮：霓虹渐变 */
.va-btn {
  position: relative;
  width: 100%;
  height: 44px;
  margin-top: 12px;
  overflow: hidden;
  font-weight: 600;
  letter-spacing: 4px;
  background: linear-gradient(90deg, #4dd0e1, #2979ff);
  border: none;
  box-shadow: 0 6px 20px rgb(41 121 255 / 35%);
  transition: all 0.2s;
}

.va-btn:hover {
  background: linear-gradient(90deg, #4dd0e1, #2979ff);
  box-shadow:
    0 8px 24px rgb(77 208 225 / 50%),
    0 0 20px rgb(77 208 225 / 30%);
  transform: translateY(-1px);
}

.va-btn::after {
  position: absolute;
  inset: 0;
  content: "";
  background: radial-gradient(circle at var(--x, 50%) var(--y, 50%), rgb(255 255 255 / 35%), transparent 60%);
  opacity: 0;
  transition: opacity 0.3s;
}

.va-btn:hover::after {
  opacity: 1;
}

/* 底部版权 */
.va-footer {
  position: absolute;
  right: 0;
  bottom: 14px;
  left: 0;
  z-index: 5;
  font-size: 12px;
  color: rgb(168 192 214 / 55%);
  text-align: center;
}

/* 语言下拉样式（保留原有） */
.translation {
  :deep(.el-dropdown-menu__item) {
    padding: 5px 40px;
  }
}

/* 响应式 */
@media screen and (width <= 1180px) {
  .va-main {
    grid-template-columns: 1fr;
    gap: 20px;
    padding: 0 24px;
  }

  .va-brand {
    display: none;
  }

  .va-card {
    justify-self: center;
    margin: auto;
  }
}

/* 高级设置：服务器地址面板 */
.va-advanced {
  margin-top: 6px;
}

.va-advanced-toggle {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 12px;
  color: #6a8cae;
  cursor: pointer;
  user-select: none;
  transition: color 0.2s;
}

.va-advanced-toggle:hover {
  color: #4dd0e1;
}

.va-caret {
  display: inline-block;
  font-size: 11px;
  transition: transform 0.2s;
}

.va-caret-open {
  transform: rotate(90deg);
}

.va-advanced-body {
  padding-top: 10px;
  overflow: hidden;
}

.va-server-input {
  width: 100%;
}

.va-advanced-actions {
  display: flex;
  gap: 4px;
  margin-top: 6px;
}

.va-link {
  padding: 2px 4px !important;
  font-size: 12px;
  color: #4dd0e1;
}

.va-advanced-tip {
  margin: 4px 0 0;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 11px;
  color: #6a8cae;
  letter-spacing: 0.3px;
}

/* 浅色模式下自动加深卡片背景（保持视觉一致） */
html:not(.dark) .va-card {
  background: rgb(12 24 48 / 72%);
}
</style>

<!-- 自动补全弹层 teleport 到 body，scoped 无法命中，故用全局样式 -->
<style>
.va-sug {
  width: 100%;
  font-family: ui-monospace, Consolas, monospace;
  font-size: 12px;
}

.va-sug-del {
  font-size: 12px;
  color: #ef5350;
  cursor: pointer;
  opacity: 0.85;
}

.va-sug-del:hover {
  opacity: 1;
}
</style>
