<template>
  <div>
    <el-row :gutter="16">
      <el-col :span="6" v-for="(status, proto) in protocols" :key="proto">
        <el-card shadow="hover" style="margin-bottom: 16px;">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <span style="font-weight: 600; color: #4fc3f7;">{{ proto.toUpperCase() }}</span>
              <el-tag :type="status.enabled ? 'success' : 'info'" size="small">
                {{ status.enabled ? '已启用' : '未启用' }}
              </el-tag>
            </div>
          </template>
          <el-descriptions :column="1" size="small">
            <el-descriptions-item v-for="(v, k) in status" :key="k" :label="k">
              {{ typeof v === 'object' ? JSON.stringify(v) : v }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="hover" v-if="Object.keys(protocols).length === 0 && !loading">
      <el-empty description="暂无协议状态数据，请确认模拟器已启动" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { getProtocolStatus } from '../api'

const protocols = ref({})
const loading = ref(false)

const load = async () => {
  loading.value = true
  try {
    const { data } = await getProtocolStatus()
    protocols.value = data || {}
  } catch (e) {
    console.error('load protocols error', e)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
