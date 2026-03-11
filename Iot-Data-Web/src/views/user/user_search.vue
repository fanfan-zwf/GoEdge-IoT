<template>
    <el-config-provider :locale="zhCn">

        <el-popover :visible="search.visible" placement="bottom" width="510">
            <div :id="componentId">
                <el-table :data="User_data" style="width: 100%" max-height="1100px">
                    <el-table-column fixed prop="Id" label="Id" min-width="30" />
                    <el-table-column prop="Name" label="用户名" min-width="60" />
                    <el-table-column prop="Phone" label="电话" min-width="110" />
                    <el-table-column prop="Email" label="邮箱" min-width="150" />
                    <el-table-column label="操作" width="60">
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
import { User__Get_Info_Search, type User__table_interface } from '@/utils/api'
import { Search } from '@element-plus/icons-vue'

interface search_interface {
    single: boolean,
    visible: boolean,
    search: string,
    result: User__table_interface[]
}

interface Props {
    choice: (value: User__table_interface) => void;
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

const User_data: User__table_interface[] = reactive([])


// 搜索
const Search_button = () => {
    User__Get_Info_Search(search.search, "Name", 20).then((User_table: User__table_interface[]) => {
        search.visible = true
        Object.assign(User_data, User_table)
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