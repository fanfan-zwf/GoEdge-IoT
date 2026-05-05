<template>
    <el-popover :visible="search.visible" placement="bottom">
        <!-- <el-button size="primary" @click.prevent="search.visible = false" class="button">关闭</el-button> -->
        <!-- <p v-for="value in search.result">{{ value.Id }} - {{ value.Name }}</p> -->
        <div @mouseleave="handleMouseLeave">
            <el-table :data="search.result" style="width: 100%" max-height="400px">
                <el-table-column fixed prop="Id" label="Id" width="60" align="center" />
                <el-table-column prop="Name" label="名称" min-width="160" show-overflow-tooltip />
                <!-- <el-table-column prop="Label" label="标识" min-width="100" show-overflow-tooltip /> -->
                <!-- <el-table-column prop="Sn" label="Sn" width="160" align="center" /> -->
                <el-table-column prop="User_Id" label="创建用户Id" width="100" align="center" />
                <!-- <el-table-column prop="Version" label="版本" width="230" align="center" /> -->
                <!-- <el-table-column prop="Creation_Time" label="创建时间" min-width="200" show-overflow-tooltip /> -->
                <el-table-column prop="Last_Activity_Time" label="最后活动时间" width="180" align="center" />
                <el-table-column label="选择" width="80" align="center">
                    <template #default="scope">
                        <el-button link type="primary" size="small" @click.prevent="choice(scope.row)">
                            选择
                        </el-button>
                    </template>
                </el-table-column>
            </el-table>
        </div>
        <!-- <el-button @click=" search.visible=false">取消</el-button> -->
        <template #reference>
            <el-input v-model.lazy="search.search">
                <template #append><el-button @click="api_Search();"><el-icon>
                            <Search />
                        </el-icon></el-button></template>
            </el-input>
        </template>
    </el-popover>

</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import axios from "axios";
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Collector_Info__Search_Field_Blurred, type Collector_Info__table_interface } from '@/api/config_service'

// export interface api_search_interface {
//     Id: number,   // 点位id
//     Drive_Id: number,    // 驱动id唯一标识符
//     Drive_Type: string,  // 驱动类型
//     Name: string, // 点位名称
//     Group: string, // 分组
//     RW_Cancel: string, // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
//     Value_Type: string, // 输出类型
// }

interface search_interface {
    visible: boolean,
    search: string,
    result: Collector_Info__table_interface[]
    multiple: boolean
}

interface Props {
    result: (value: Collector_Info__table_interface) => void;
}


// 接收父组件传递过来的函数
const props = defineProps<Props>();

// 报警配置搜索
var search: search_interface = reactive<search_interface>({
    visible: false,
    search: "%%",
    result: [],
    multiple: false,
})

// props.pointid_result(0, "123", "123")



const choice = (row: Collector_Info__table_interface) => {
    if (search.multiple === false) {
        search.visible = false
        search.search = row.Name
    }
    props.result(row)
}


const api_Search = () => {
    Collector_Info__Search_Field_Blurred(
        { Quantity: 20, Vague: search.search }
    ).then((value_array: Collector_Info__table_interface[]) => {
        search.result.length = 0
        Object.assign(search.result, value_array)
        search.visible = true
    }).catch((error) => {
        ElMessage.error(error)
    })

}

// 鼠标离开了组件
const handleMouseLeave = (event: MouseEvent) => {
    console.log('鼠标离开了组件', event)
    search.visible = false
}

</script>

<style>
.hint {
    color: #ff0000;
    font-size: 14px;
    margin-left: 20px;
}

.button {
    width: 100%;
    height: 30px;

}


.el-popover {
    width: 1000px !important;
}

/* 屏幕宽度小于800px时，调整.el-dialog宽度为80% */
@media (max-width: 800px) {
    .el-popover {
        width: 90% !important;
    }
}

/* 更小的手机 < 480px */
@media (max-width: 480px) {
    .el-popover {
        width: 460px !important;
    }
}
</style>