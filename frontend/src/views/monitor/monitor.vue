<template>
    <el-table :data="tableData" style="width: 100%">
        <el-table-column fixed prop="Id" label="ID" width="40" />
        <el-table-column prop="Tag" label="点位标识" min-width="120" />
        <el-table-column prop="Msg" label="状态" width="120" />
        <el-table-column prop="Type" label="类型" width="120" />
        <el-table-column prop="Value" label="值" width="120" />
        <el-table-column prop="Time" label="时间" width="400" />
        <el-table-column prop="Last_Value" label="上次值" width="120" />
        <el-table-column prop="Last_Time" label="上次时间" width="400" />
        <el-table-column fixed="right" label="操作" min-width="120">
            <template #default>
                <el-button link type="primary" size="small" @click="handleClick">
                    Detail
                </el-button>
                <el-button link type="primary" size="small">Edit</el-button>
            </template>
        </el-table-column>
    </el-table>
</template>

<script setup lang="ts">
import { reactive, ref, onUnmounted } from "vue"
import axios from "axios";
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { http_Front_url, ws_Front_url } from '@/typer/index'
import { Points_length } from '@/typer/api'
import { } from '@/typer/api'
const handleClick = () => {
    console.log('click')
}

const tableData = [
    {
        date: '2016-05-03',
        name: 'Tom',
        state: 'California',
        city: 'Los Angeles',
        address: 'No. 189, Grove St, Los Angeles',
        zip: 'CA 90036',
        tag: 'Home',
    },
    {
        date: '2016-05-02',
        name: 'Tom',
        state: 'California',
        city: 'Los Angeles',
        address: 'No. 189, Grove St, Los Angeles',
        zip: 'CA 90036',
        tag: 'Office',
    },
    {
        date: '2016-05-04',
        name: 'Tom',
        state: 'California',
        city: 'Los Angeles',
        address: 'No. 189, Grove St, Los Angeles',
        zip: 'CA 90036',
        tag: 'Home',
    },
    {
        date: '2016-05-01',
        name: 'Tom',
        state: 'California',
        city: 'Los Angeles',
        address: 'No. 189, Grove St, Los Angeles',
        zip: 'CA 90036',
        tag: 'Office',
    },
]
const props = {
    expandTrigger: 'hover' as const,
}

interface monitor_value_interface {
    PC_Id: string, // 设备id
    Points_Id: number, // 点位id
    Comments: string, // 状态
    Value_Type: string, // 输出类型
    Value: any, // 值
    Time: string, // 输出类型
    Last_Value: any, // 值
    Last_Time: string,// 时间戳
}

interface monitor_info_interface {
    Id: number,   // 点位id
    Name: string, // 名称
    Group: string, // 组
    RW_Cancel: string, // 读写方式
    Value_Type: string, // 输出类型
    Realtime_value: monitor_value_interface
}

const monitor = reactive({
    points: [] as monitor_info_interface[],
    ws: null as WebSocket | null,
})


const write_value_Row = function (scope: any) {
    console.log(scope.row)
    ElMessage({
        message: '写值功能正在开发中',
        type: 'info',
    })
}




</script>

<style>
.table {
    text-align: center;
}
</style>
