<template>
    <el-config-provider :locale="zhCn">

        <el-popover :visible="search.visible" placement="bottom" width="500">
            <div :id="componentId">
                <el-table :data="authority_data" style="width: 100%" max-height="800px">
                    <el-table-column fixed prop="Id" label="Id" max-width="8" />
                    <el-table-column fixed prop="Name" label="名称" max-width="50" />
                    <el-table-column fixed prop="Theme" label="主题" max-width="70" />
                    <el-table-column fixed prop="Explain" label="说明" max-width="70" />
                    <el-table-column label="操作" width="70">
                        <template #default="scope">
                            <el-button link type="primary" size="small" @click.prevent="Choice_button(scope)">
                                选择
                            </el-button>
                        </template>
                    </el-table-column>
                </el-table>
            </div>
            <template #reference>
                <el-input v-model="search.search" placeholder="输入权限名称">
                    <template #append>
                        <el-button @click="Search_button" :icon="Search" />
                    </template>
                </el-input>
            </template>


        </el-popover>

    </el-config-provider>
</template>

<script setup lang="ts">
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Authority__Search, type Authority__table_interface } from '@/typer/api'
import { Search } from '@element-plus/icons-vue'

interface search_interface {
    single: boolean,
    visible: boolean,
    search: string,
    result: Authority__table_interface[]
}

interface Props {
    choice: (value: Authority__table_interface) => void;
}


// 接收父组件传递过来的函数
const props = defineProps<Props>();

//  搜索
var search: search_interface = reactive<search_interface>({
    single: false,
    visible: false,
    search: "%%",
    result: []
})

const authority_data: Authority__table_interface[] = reactive([])


// 搜索
const Search_button = () => {
    Authority__Search(search.search, "Name", 20).then((Authority_table: Authority__table_interface[]) => {
        search.visible = true
        Object.assign(authority_data, Authority_table)
    })
}

// 选择
const Choice_button = (scope: any) => {
    props.choice(scope.row)
    if (!search.single) {
        search.visible = false
    }
}

// // 鼠标离开了组件
// const handleMouseLeave = (event: MouseEvent) => {
//     console.log('鼠标离开了组件', event)
//     search.visible = false
// }


// 生成随机 ID
const generateRandomId = () => {
    return `component_${Date.now()}_${Math.random().toString(36).substr(2, 6)}`
}

const componentId = ref(generateRandomId())

const handleClickOutside = (event: MouseEvent) => {
    const element = document.getElementById(componentId.value)
    if (element && !element.contains(event.target as Node)) {
        // 关闭逻辑
        search.visible = false
    }
}

onMounted(() => {
    document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
    document.removeEventListener('click', handleClickOutside)
}) 
</script>

<style scoped>
.hint {
    color: #ff0000;
    font-size: 14px;
    margin-left: 20px;
}

.button {
    width: 100%;
    height: 30px;

}
</style>