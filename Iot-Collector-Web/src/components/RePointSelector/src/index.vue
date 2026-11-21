<script setup lang="ts">
import { ref, computed, watch, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { Search, Folder, Document } from "@element-plus/icons-vue";
import { configApi } from "@/api/config";

interface PointItem {
  Id: number;
  Tag: string;
  Config?: string;
  Drive_Id?: number;
  [key: string]: any;
}

interface TreeNode {
  id: number;
  name: string;
  path: string;
  nodeType: number;
  hasChildren: number;
  pointId?: number;
}

const props = withDefaults(
  defineProps<{
    /** 对话框可见性 */
    visible: boolean;
    /** 是否多选 */
    multiple?: boolean;
    /** 已选中的点位 ID（单选为 number，多选为 number[]） */
    selected?: number | number[];
  }>(),
  {
    multiple: false,
    selected: () => []
  }
);

const emit = defineEmits<{
  (e: "update:visible", val: boolean): void;
  (e: "confirm", points: PointItem[]): void;
}>();

const { t } = useI18n();

// ---- Tree ----
const treeRef = ref<any>(null);
const treeKey = ref(0);
const treeProps = {
  label: "name",
  isLeaf: (data: TreeNode) => data.hasChildren !== 1
};

const loadNode = async (node: any, resolve: (data: TreeNode[]) => void) => {
  const tagPath = node.level === 0 ? "" : node.data.path;
  try {
    const res: any = await configApi.tagTree.children({ tag_path: tagPath });
    const data = res?.Data || {};
    const folders: TreeNode[] = data["子路径"] || [];
    const files: TreeNode[] = data["文件"] || [];
    const merged = new Map<number, TreeNode>();
    for (const n of [...folders, ...files]) merged.set(n.id, n);
    resolve([...merged.values()]);
  } catch {
    resolve([]);
  }
};

const activeTreePath = ref("");

const handleTreeNodeClick = async (data: TreeNode) => {
  if (activeTreePath.value === data.path) {
    activeTreePath.value = "";
  } else {
    activeTreePath.value = data.path;
  }
  await loadPointsByTree();
};

const loadPointsByTree = async () => {
  if (!activeTreePath.value) {
    pointList.value = [];
    return;
  }
  treeLoading.value = true;
  try {
    const res: any = await configApi.tagTree.points({
      tag_path: activeTreePath.value
    });
    pointList.value = res.Data || [];
  } catch {
    pointList.value = [];
  } finally {
    treeLoading.value = false;
  }
};

// ---- Search ----
const searchKeyword = ref("");
const searchLoading = ref(false);
const pointList = ref<PointItem[]>([]);
const treeLoading = ref(false);

const handleSearch = async () => {
  if (!searchKeyword.value.trim()) {
    pointList.value = [];
    return;
  }
  searchLoading.value = true;
  try {
    const res: any = await configApi.point.search({
      tag: `%${searchKeyword.value.trim()}%`,
      quantity: 50
    });
    pointList.value = res.Data || [];
  } catch {
    pointList.value = [];
  } finally {
    searchLoading.value = false;
  }
};

// ---- Selection ----
const tableRef = ref<any>(null);
const selectedPoints = ref<PointItem[]>([]);

/** 初始化已选（打开对话框时） */
watch(
  () => props.visible,
  val => {
    if (val) {
      // 重置状态
      searchKeyword.value = "";
      activeTreePath.value = "";
      pointList.value = [];
      // 回填已选
      const ids = props.multiple
        ? (props.selected as number[]) || []
        : [props.selected as number].filter(Boolean);
      selectedPoints.value = ids.map(id => ({ Id: id, Tag: `#${id}` } as PointItem));
      treeKey.value += 1;
    }
  }
);

const isPointSelected = (row: PointItem) => {
  return selectedPoints.value.some(p => p.Id === row.Id);
};

const toggleSelect = (row: PointItem) => {
  if (props.multiple) {
    const idx = selectedPoints.value.findIndex(p => p.Id === row.Id);
    if (idx >= 0) {
      selectedPoints.value.splice(idx, 1);
    } else {
      selectedPoints.value.push(row);
    }
  } else {
    // 单选：替换
    selectedPoints.value = [row];
  }
};

const removeSelected = (id: number) => {
  selectedPoints.value = selectedPoints.value.filter(p => p.Id !== id);
};

// ---- Confirm / Cancel ----
const handleConfirm = () => {
  emit("confirm", selectedPoints.value);
  close();
};

const close = () => {
  emit("update:visible", false);
};

// ---- Display helpers ----
const pointLabel = (p: PointItem) => {
  return `[${p.Id}] ${p.Tag || p.Config || ""}`;
};
</script>

<template>
  <el-dialog
    :model-value="visible"
    :title="t('pointSelector.title')"
    width="780px"
    destroy-on-close
    @update:model-value="emit('update:visible', $event)"
    @close="close"
  >
    <div class="ps-layout">
      <!-- 左侧：树 -->
      <div class="ps-tree">
        <el-scrollbar height="360px">
          <el-tree
            ref="treeRef"
            :key="treeKey"
            :load="loadNode"
            lazy
            node-key="id"
            :props="treeProps"
            highlight-current
            :expand-on-click-node="false"
            @node-click="handleTreeNodeClick"
          >
            <template #default="{ data }">
              <span class="tree-node">
                <el-icon class="node-icon">
                  <Folder v-if="data.nodeType !== 2" />
                  <Document v-else />
                </el-icon>
                <span class="node-label">{{ data.name }}</span>
              </span>
            </template>
          </el-tree>
        </el-scrollbar>
      </div>

      <!-- 右侧：搜索 + 列表 -->
      <div class="ps-list">
        <div class="ps-search">
          <el-input
            v-model="searchKeyword"
            :placeholder="t('pointSelector.searchPlaceholder')"
            clearable
            @keyup.enter="handleSearch"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-button type="primary" @click="handleSearch">{{ t("common.search") }}</el-button>
        </div>

        <el-scrollbar v-loading="treeLoading || searchLoading" height="310px" class="ps-table-wrap">
          <div v-if="pointList.length === 0" class="ps-empty">
            {{ t("pointSelector.emptyHint") }}
          </div>
          <div
            v-for="row in pointList"
            :key="row.Id"
            :class="['ps-item', { 'is-selected': isPointSelected(row) }]"
            @click="toggleSelect(row)"
          >
            <el-checkbox v-if="multiple" :model-value="isPointSelected(row)" @click.stop @change="toggleSelect(row)" />
            <el-radio v-else :model-value="selectedPoints[0]?.Id" :value="row.Id" @click.stop="toggleSelect(row)">
              <span />
            </el-radio>
            <span class="ps-item-label">{{ pointLabel(row) }}</span>
          </div>
        </el-scrollbar>
      </div>
    </div>

    <!-- 已选区域 -->
    <div v-if="selectedPoints.length > 0" class="ps-selected">
      <span class="ps-selected-label">{{ t("pointSelector.selected") }} ({{ selectedPoints.length }}):</span>
      <el-tag
        v-for="p in selectedPoints"
        :key="p.Id"
        closable
        size="small"
        class="ps-selected-tag"
        @close="removeSelected(p.Id)"
      >
        {{ p.Tag || `#${p.Id}` }}
      </el-tag>
    </div>

    <template #footer>
      <el-button @click="close">{{ t("common.cancel") }}</el-button>
      <el-button type="primary" :disabled="selectedPoints.length === 0" @click="handleConfirm">
        {{ t("common.confirm") }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.ps-layout {
  display: flex;
  gap: 12px;
}

.ps-tree {
  flex-shrink: 0;
  width: 220px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}

.ps-list {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}

.ps-search {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.ps-table-wrap {
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}

.ps-empty {
  padding: 24px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  text-align: center;
}

.ps-item {
  display: flex;
  gap: 8px;
  align-items: center;
  padding: 6px 12px;
  cursor: pointer;
  transition: background 0.15s;
}

.ps-item:hover {
  background: var(--el-fill-color-light);
}

.ps-item.is-selected {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.ps-item-label {
  font-size: 13px;
  color: var(--el-text-color-primary);
}

.ps-item.is-selected .ps-item-label {
  color: var(--el-color-primary);
}

.ps-selected {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
  padding-top: 10px;
  margin-top: 10px;
  border-top: 1px dashed var(--el-border-color-lighter);
}

.ps-selected-label {
  font-size: 13px;
  color: var(--el-text-color-regular);
}

.ps-selected-tag {
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tree-node {
  display: flex;
  align-items: center;
  font-size: 13px;
}

.node-icon {
  margin-right: 4px;
}
</style>
