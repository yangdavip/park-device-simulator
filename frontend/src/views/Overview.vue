<template>
  <div>
    <!-- 统计卡片 -->
    <el-row :gutter="16" class="stat-row">
      <el-col :span="6" v-for="card in statCards" :key="card.label">
        <el-card shadow="hover">
          <el-statistic :title="card.label" :value="card.value" :suffix="card.suffix" />
        </el-card>
      </el-col>
    </el-row>

    <!-- 图表区 -->
    <el-row :gutter="16" style="margin-top: 16px;">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>设备系统分布</template>
          <v-chart :option="systemChartOption" style="height: 300px;" autoresize />
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>设备状态</template>
          <v-chart :option="statusChartOption" style="height: 300px;" autoresize />
        </el-card>
      </el-col>
    </el-row>

    <!-- 最新告警 -->
    <el-card shadow="hover" style="margin-top: 16px;">
      <template #header>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <span>最新告警</span>
          <el-tag size="small" type="danger">{{ alarms.length }} 条</el-tag>
        </div>
      </template>
      <el-table :data="alarms" style="width: 100%" size="small" :max-height="300">
        <el-table-column prop="device_id" label="设备ID" width="200" />
        <el-table-column prop="type" label="告警类型" width="180" />
        <el-table-column prop="level" label="级别" width="100">
          <template #default="{ row }">
            <el-tag :type="levelType(row.level)" size="small">{{ row.level }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="message" label="描述" />
        <el-table-column label="时间" width="180">
          <template #default="{ row }">{{ formatTime(row.ts) }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart, BarChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { getStats, getAlarms } from '../api'
import dayjs from 'dayjs'

use([CanvasRenderer, PieChart, BarChart, TitleComponent, TooltipComponent, LegendComponent, GridComponent])

const stats = ref(null)
const alarms = ref([])

const statCards = computed(() => {
  const s = stats.value || {}
  return [
    { label: '设备总数', value: s.total_devices || 0, suffix: '台' },
    { label: '在线设备', value: s.online_devices || 0, suffix: '台' },
    { label: '在线率', value: ((s.online_rate || 0)).toFixed(1), suffix: '%' },
    { label: '当前场景', value: s.current_scenario || '-', suffix: '' },
  ]
})

const systemChartOption = computed(() => ({
  tooltip: { trigger: 'item' },
  legend: { bottom: 0, textStyle: { color: '#8892b0' } },
  series: [{
    type: 'pie',
    radius: ['40%', '70%'],
    data: Object.entries(stats.value?.system_stats || {}).map(([name, value]) => ({ name, value })),
    itemStyle: { borderColor: '#0a0e27', borderWidth: 2 },
    label: { color: '#e0e0e0' },
  }],
  color: ['#4fc3f7', '#66bb6a', '#ffa726', '#ef5350', '#ab47bc', '#26c6da', '#ffca28', '#8d6e63', '#78909c', '#ec407a'],
}))

const statusChartOption = computed(() => {
  const s = stats.value || {}
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: '10%', right: '10%', top: '10%', bottom: '15%' },
    xAxis: { type: 'category', data: ['在线', '离线'], axisLabel: { color: '#8892b0' } },
    yAxis: { type: 'value', axisLabel: { color: '#8892b0' } },
    series: [{
      type: 'bar',
      data: [
        { value: s.online_devices || 0, itemStyle: { color: '#66bb6a' } },
        { value: s.offline_devices || 0, itemStyle: { color: '#ef5350' } },
      ],
      barWidth: '40%',
      label: { show: true, position: 'top', color: '#e0e0e0' },
    }],
  }
})

const levelType = (level) => {
  const map = { critical: 'danger', major: 'danger', warning: 'warning', minor: 'info' }
  return map[level] || 'info'
}

const formatTime = (ts) => dayjs(ts).format('MM-DD HH:mm:ss')

let pollTimer = null

const poll = async () => {
  try {
    const [statsRes, alarmsRes] = await Promise.all([getStats(), getAlarms()])
    stats.value = statsRes.data
    alarms.value = (alarmsRes.data.alarms || []).slice(0, 10)
  } catch (e) {
    console.error('poll error', e)
  }
}

onMounted(() => {
  poll()
  pollTimer = setInterval(poll, 5000)
})

onUnmounted(() => clearInterval(pollTimer))
</script>

<style scoped>
.stat-row { margin-bottom: 0; }
</style>
