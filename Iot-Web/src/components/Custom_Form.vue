<template>
    <template v-for="(field, idx) in fieldRules" :key="idx">
        <el-form-item :prop="field.prop" :label="field.label" v-if="isFieldVisible(field)" size="large"
            :error="fieldError[field.prop]">
            <el-select v-if="field.type === 'select'" v-model="UpdateItem[field.prop]" style="width: 100%" size="large"
                :placeholder="field.placeholder || '请选择'" @change="validateField(field)">
                <el-option v-for="opt in field.options" :key="opt.value" :label="opt.label" :value="opt.value" />
                <template #append v-if="field.unitType !== null && field.unitType !== undefined">{{ field.unitType
                }}</template>
            </el-select>

            <el-input v-else-if="!field.type || field.type === 'string'" v-model="UpdateItem[field.prop]"
                style="width: 100%" size="large" :placeholder="field.placeholder || '请输入'" clearable
                @input="validateField(field)">
                <template #append v-if="field.unitType !== null && field.unitType !== undefined">{{ field.unitType
                }}</template>
            </el-input>

            <el-input v-else-if="field.type === 'number'" v-model="UpdateItem[field.prop]" type="number"
                style="width: 100%" size="large" :placeholder="field.placeholder || '请输入数字'" clearable
                @input="validateField(field)">
                <template #append v-if="field.unitType !== null && field.unitType !== undefined">{{ field.unitType
                }}</template>
            </el-input>

            <el-input v-else-if="field.type === 'unit'" v-model="UpdateItem[field.prop]" type="number"
                style="width: 100%" size="large" :placeholder="field.placeholder || '请输入数值'" clearable
                @input="validateField(field)">
                <template #append v-if="field.unitType !== null && field.unitType !== undefined">{{ field.unitType
                }}</template>
            </el-input>
        </el-form-item>
    </template>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

export interface DynamicFieldItem {
    prop: string
    label: string
    type?: 'string' | 'number' | 'select' | 'unit'
    options?: { label: string; value: any }[]
    unitType?: string
    condition?: (item: any) => boolean
    regex?: RegExp
    message?: string
    placeholder?: string
    hidden?: boolean | (() => boolean)
}

const props = defineProps({
    modelValue: String,
    fieldRules: {
        type: Array as () => DynamicFieldItem[],
        required: true
    },
    UpdateItem: {
        type: Object,
        required: true
    }
})

const emit = defineEmits(['update:modelValue'])
const fieldError = ref<Record<string, string>>({})

// 判断是否隐藏
function isHidden(field: DynamicFieldItem): boolean {
    if (typeof field.hidden === 'function') {
        return !!field.hidden()
    }
    return !!field.hidden
}

// 是否显示
function isFieldVisible(field: DynamicFieldItem) {
    if (isHidden(field)) return false
    if (field.condition) {
        return field.condition(props.UpdateItem)
    }
    return true
}

// 验证
function validateField(field: DynamicFieldItem) {
    if (!isFieldVisible(field)) {
        fieldError.value[field.prop] = ''
        return
    }
    const val = props.UpdateItem[field.prop] ?? ''
    if (field.regex) {
        const isValid = field.regex.test(val + '')
        fieldError.value[field.prop] = isValid ? '' : field.message || '格式错误'
    } else {
        fieldError.value[field.prop] = ''
    }
}

// 监听隐藏 → 清空
watch(
    () => props.fieldRules.map(f => isHidden(f)),
    () => {
        props.fieldRules.forEach(f => {
            if (isHidden(f)) {
                props.UpdateItem[f.prop] = ''
            }
        })
    },
    { deep: true, immediate: true }
)

// 解析
watch(
    () => props.modelValue,
    (val) => {
        if (!val) return
        const arr = val.split(';')
        props.fieldRules.forEach((f, i) => {
            let value = arr[i] ?? ''
            // 👇 只有 unitType 存在时才去除单位字符
            if (f.type === 'unit' && f.unitType !== undefined) {
                value = value.replace(/\D/g, '')
            }
            props.UpdateItem[f.prop] = value
            validateField(f)
        })
    },
    { immediate: true }
)

// 拼接（unitType = undefined 不拼接单位）
watch(
    () => props.fieldRules.map(f => props.UpdateItem[f.prop]),
    () => {
        const str = props.fieldRules
            .map(f => {
                if (isHidden(f)) return ''
                let val = props.UpdateItem[f.prop] ?? ''

                // 👇 核心：unitType 有值才拼接，undefined 就纯数字
                if (f.type === 'unit' && val && f.unitType !== undefined) {
                    val = `${val}${f.unitType}`
                }

                return val
            })
            .join(';')

        emit('update:modelValue', str)
    },
    { deep: true }
)
</script>