<template>
    <el-table :data="monitor_data" style="width: 100%">
        <el-table-column fixed prop="Info.Id" label="ID" width="40" />
        <el-table-column prop="Info.Tag" label="点位标识" min-width="120" />
        <el-table-column prop="Value.Msg" label="状态" width="120" />
        <el-table-column prop="Info.Type" label="类型" width="120" />
        <el-table-column prop="Value.Value" label="值" width="120" />
        <el-table-column prop="Value.Time" label="时间" width="400" />
        <el-table-column prop="Value.Last_Value" label="上次值" width="120" />
        <el-table-column prop="Value.Last_Time" label="上次时间" width="400" />
        <el-table-column fixed="right" label="操作" min-width="120">
            <template #default>
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
// import { Points_length } from '@/utils/api'
import { } from '@/utils/api'

interface monitor_value_interface {
    Id: number,   // 点位id
    Tag: string,  // 点位标识
    Msg: string,  // 状态
    Type: string,  // 值类型

    Value: string,  // 值
    Time: string,// 时间戳

    Last_Value: string, // 值
    Last_Time: string,// 时间戳
}

interface monitor_info_interface {
    Id: number,   // 点位id
    Drive_Id: number,  // 驱动id唯一标识符
    Tag: number, // 点位标识
    Drive_Type: number, // 驱动类型
    Description: number, // 点位描述
    RW_Cancel: number, // 点位读写方式 读写方式 N:禁用  R:只读  W:只写  R/W:读写
    Value_Type: number, // 输出类型
    Config: number,
}

interface monitor_interface {
    Info: monitor_info_interface,
    Value: monitor_value_interface
}


let monitor_data: monitor_interface[] = reactive<monitor_interface[]>([])




 


</script>

<style>
.table {
    text-align: center;
}
</style>
