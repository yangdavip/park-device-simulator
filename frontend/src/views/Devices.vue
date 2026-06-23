<template>
  <div>
    <!-- 筛选栏 -->
    <el-card shadow="hover" style="margin-bottom: 16px;">
      <el-form :inline="true" :model="filters">
        <el-form-item label="系统">
          <el-select v-model="filters.system" placeholder="全部系统" clearable style="width: 150px;">
            <el-option v-for="s in systems" :key="s" :label="s" :value="s" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="filters.status" placeholder="全部状态" clearable style="width: 120px;">
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
          </el-select>
        </el-form-item>
        <el-form-item label="搜索">
          <el-input v-model="filters.keyword" placeholder="设备ID/类型" clearable style="width: 200px;" @keyup.enter="loadDevices" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadDevices">查询</el-button>
          <el-button @click="resetFilters">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 设备表格 -->
    <el-card shadow="hover">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>设备列表（共 {{ total }} 台）</span>
          <el-button size="small" @click="loadDevices" :icon="Refresh">刷新</el-button>
        </div>
      </template>
      <el-table :data="devices" style="width: 100%" size="small" v-loading="loading" :max-height="600">
        <el-table-column prop="id" label="设备ID" width="220" />
        <el-table-column prop="type" label="类型" width="160">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="system" label="系统" width="120" />
        <el-table-column prop="protocol" label="协议" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.protocol }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.status === 'online' ? 'success' : 'danger'" size="small">
              {{ row.status === 'online' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最新数据" min-width="300">
          <template #default="{ row }">
            <div v-if="row.last_data">
              <el-tag v-for="(v, k) in row.last_data" :key="k" size="small" style="margin: 2px;">
                {{ k }}: {{ formatVal(v) }}
              </el-tag>
            </div>
            <span v-else style="color: #8892b0;">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button size="small" link @click="showDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 详情弹窗 -->
    <el-dialog v-model="detailVisible" :title="`设备详情 - ${currentDevice?.id}`" width="600px">
      <el-descriptions :column="2" border v-if="currentDevice">
        <el-descriptions-item label="设备ID">{{ currentDevice.id }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ currentDevice.type }}</el-descriptions-item>
        <el-descriptions-item label="系统">{{ currentDevice.system }}</el-descriptions-item>
        <el-descriptions-item label="协议">{{ currentDevice.protocol }}</el-descriptions-item>
        <el-descriptions-item label="状态">{{ currentDevice.status }}</el-descriptions-item>
        <el-descriptions-item label="楼层">{{ currentDevice.metadata?.floor || '-' }}</el-descriptions-item>
      </el-descriptions>
      <el-divider />
      <div v-if="currentDevice?.last_data">
        <h4 style="color: #4fc3f7; margin-bottom: 12px;">最新数据</h4>
        <el-table :data="dataEntries" size="small" border>
          <el-table-column prop="key" label="数据点" width="200" />
          <el-table-column prop="value" label="值" />
        </el-table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { getDevices } from '../api'

const devices = ref([])
const total = ref(0)
const loading = ref(false)
const detailVisible = ref(false)
const currentDevice = ref(null)

const filters = reactive({
  system: '',
  status: '',
  keyword: ''
})

const systems = ['bas', 'lighting', 'security', 'access', 'fire', 'parking', 'energy', 'environment', 'elevator', 'broadcast']

const dataEntries = computed(() => {
  if (!currentDevice.value?.last_data) return []
  return Object.entries(currentDevice.value.last_data).map(([key, value]) => ({ key, value: String(value) }))
})

const formatVal = (v) => {
  if (typeof v === 'number') return v.toFixed(2)
  return String(v)
}

const loadDevices = async () => {
  loading.value = true
  try {
    const params = {}
    if (filters.system) params.system = filters.system
    if (filters.status) params.status = filters.status
    if (filters.keyword) params.keyword = filters.keyword
    const { data } = await getDevices(params)
    devices.value = data.devices || []
    total.value = data.total || 0
  } catch (e) {
    console.error('loadDevices error', e)
  } finally {
    loading.value = false
  }
}

const resetFilters = () => {
  filters.system = ''
  filters.status = ''
  filters.keyword = ''
  loadDevices()
}

const showDetail = (row) => {
  currentDevice.value = row
  detailVisible.value = true
}

onMounted(loadDevices)
</script>
