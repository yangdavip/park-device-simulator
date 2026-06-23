<template>
  <div>
    <!-- 告警统计 -->
    <el-row :gutter="16" style="margin-bottom: 16px;">
      <el-col :span="6" v-for="card in alarmStats" :key="card.label">
        <el-card shadow="hover">
          <el-statistic :title="card.label" :value="card.value" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 告警列表 -->
    <el-card shadow="hover">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>告警列表（共 {{ alarms.length }} 条）</span>
          <div>
            <el-select v-model="levelFilter" placeholder="全部级别" clearable size="small" style="width: 120px; margin-right: 8px;">
              <el-option label="严重" value="critical" />
              <el-option label="警告" value="warning" />
              <el-option label="次要" value="minor" />
            </el-select>
            <el-button size="small" @click="loadAlarms" :icon="Refresh">刷新</el-button>
          </div>
        </div>
      </template>
      <el-table :data="filteredAlarms" style="width: 100%" size="small" v-loading="loading" :max-height="600">
        <el-table-column prop="device_id" label="设备ID" width="220" />
        <el-table-column prop="type" label="告警类型" width="180" />
        <el-table-column prop="level" label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="levelType(row.level)" size="small">{{ levelLabel(row.level) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="描述" min-width="300" />
        <el-table-column label="时间" width="180">
          <template #default="{ row }">{{ formatTime(row.ts) }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { getAlarms } from '../api'
import dayjs from 'dayjs'

const alarms = ref([])
const loading = ref(false)
const levelFilter = ref('')

const alarmStats = computed(() => {
  const list = alarms.value
  return [
    { label: '告警总数', value: list.length },
    { label: '严重', value: list.filter(a => a.level === 'critical').length },
    { label: '警告', value: list.filter(a => a.level === 'warning').length },
    { label: '次要', value: list.filter(a => a.level === 'minor').length },
  ]
})

const filteredAlarms = computed(() => {
  if (!levelFilter.value) return alarms.value
  return alarms.value.filter(a => a.level === levelFilter.value)
})

const levelType = (level) => {
  const map = { critical: 'danger', major: 'danger', warning: 'warning', minor: 'info' }
  return map[level] || 'info'
}

const levelLabel = (level) => {
  const map = { critical: '严重', major: '严重', warning: '警告', minor: '次要' }
  return map[level] || level
}

const formatTime = (ts) => dayjs(ts).format('YYYY-MM-DD HH:mm:ss')

const loadAlarms = async () => {
  loading.value = true
  try {
    const { data } = await getAlarms()
    alarms.value = data.alarms || []
  } catch (e) {
    console.error('loadAlarms error', e)
  } finally {
    loading.value = false
  }
}

onMounted(loadAlarms)
</script>
