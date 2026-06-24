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
          <div>
            <el-button type="success" size="small" @click="showAddDialog" :icon="Plus">添加设备</el-button>
            <el-button size="small" @click="loadDevices" :icon="Refresh">刷新</el-button>
          </div>
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
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button size="small" link @click="showDetail(row)">详情</el-button>
            <el-button size="small" link type="danger" @click="confirmDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加设备弹窗 -->
    <el-dialog v-model="addVisible" title="添加模拟设备" width="520px">
      <el-form :model="addForm" label-width="80px" :rules="addRules" ref="addFormRef">
        <el-form-item label="系统" prop="system">
          <el-select v-model="addForm.system" placeholder="选择系统" style="width: 100%;" @change="onSystemChange">
            <el-option v-for="s in systems" :key="s" :label="s" :value="s" />
          </el-select>
        </el-form-item>
        <el-form-item label="设备类型" prop="type">
          <el-select v-model="addForm.type" placeholder="选择设备类型" style="width: 100%;" :disabled="!addForm.system">
            <el-option v-for="t in availableTypes" :key="t" :label="t" :value="t" />
          </el-select>
        </el-form-item>
        <el-form-item label="协议" prop="protocol">
          <el-select v-model="addForm.protocol" placeholder="选择协议" style="width: 100%;">
            <el-option label="MQTT" value="mqtt" />
            <el-option label="HTTP" value="http" />
            <el-option label="Modbus" value="modbus" />
            <el-option label="OPC UA" value="opcua" />
          </el-select>
        </el-form-item>
        <el-form-item label="楼宇">
          <el-input v-model="addForm.building" placeholder="默认 B001" />
        </el-form-item>
        <el-form-item label="楼层">
          <el-input-number v-model="addForm.floor" :min="1" :max="50" />
        </el-form-item>
        <el-form-item label="位置">
          <el-input v-model="addForm.location" placeholder="如：大堂、机房（可选）" />
        </el-form-item>
        <el-form-item label="设备ID">
          <el-input v-model="addForm.custom_id" placeholder="留空自动生成（类型-楼宇-序号）" />
        </el-form-item>
        <el-form-item label="数量">
          <el-input-number v-model="addForm.count" :min="1" :max="50" />
          <span style="margin-left: 8px; color: #909399; font-size: 12px;">批量创建数量</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addVisible = false">取消</el-button>
        <el-button type="primary" @click="submitAdd" :loading="addLoading">创建</el-button>
      </template>
    </el-dialog>

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
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Plus } from '@element-plus/icons-vue'
import { getDevices, getDeviceData, createDevice, deleteDevice } from '../api'

const devices = ref([])
const total = ref(0)
const loading = ref(false)
const detailVisible = ref(false)
const currentDevice = ref(null)

// 添加设备相关
const addVisible = ref(false)
const addLoading = ref(false)
const addFormRef = ref(null)
const addForm = reactive({
  system: '',
  type: '',
  protocol: 'mqtt',
  building: 'B001',
  floor: 1,
  location: '',
  custom_id: '',
  count: 1
})
const addRules = {
  system: [{ required: true, message: '请选择系统', trigger: 'change' }],
  type: [{ required: true, message: '请选择设备类型', trigger: 'change' }],
  protocol: [{ required: true, message: '请选择协议', trigger: 'change' }]
}

// 系统 → 设备类型映射
const systemDeviceMap = {
  bas: ['ahu', 'fcu', 'fau', 'chiller', 'cooling_tower', 'pump', 'water_tank', 'vent_fan', 'heat_exchanger'],
  lighting: ['lighting_circuit', 'lux_sensor', 'lamp_controller', 'scene_panel'],
  security: ['ip_camera', 'ptz_camera', 'video_analyzer', 'infrared_beam', 'electric_fence'],
  access: ['access_controller', 'face_terminal', 'visitor_kiosk', 'turnstile'],
  fire: ['smoke_detector', 'temp_detector', 'manual_call_point', 'fire_hydrant', 'sprinkler_pump', 'fire_door'],
  parking: ['lpr_camera', 'geomagnetic', 'ultrasonic_sensor', 'guide_screen', 'charging_pile'],
  energy: ['power_meter', 'water_meter', 'gas_meter', 'heat_meter', 'pv_inverter', 'battery_storage'],
  environment: ['temp_humidity_sensor', 'pm25_sensor', 'co2_sensor', 'noise_sensor', 'gas_sensor', 'weather_station'],
  elevator: ['elevator_controller', 'escalator_controller'],
  broadcast: ['broadcast_terminal', 'emergency_broadcast']
}

const availableTypes = computed(() => systemDeviceMap[addForm.system] || [])

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

const showDetail = async (row) => {
  currentDevice.value = row
  detailVisible.value = true
  try {
    const { data } = await getDeviceData(row.id)
    if (data.last_data) {
      currentDevice.value = { ...row, last_data: data.last_data }
    }
  } catch (e) {
    console.error('loadDeviceData error', e)
  }
}

// 添加设备
const showAddDialog = () => {
  Object.assign(addForm, {
    system: '', type: '', protocol: 'mqtt', building: 'B001',
    floor: 1, location: '', custom_id: '', count: 1
  })
  addVisible.value = true
}

const onSystemChange = () => {
  addForm.type = ''
}

const submitAdd = async () => {
  if (!addFormRef.value) return
  await addFormRef.value.validate(async (valid) => {
    if (!valid) return
    addLoading.value = true
    try {
      const count = addForm.count
      let successCount = 0
      let lastError = ''
      for (let i = 0; i < count; i++) {
        const payload = {
          system: addForm.system,
          type: addForm.type,
          protocol: addForm.protocol,
          building: addForm.building,
          floor: addForm.floor,
          location: addForm.location
        }
        // 仅第一个使用自定义 ID
        if (i === 0 && addForm.custom_id) {
          payload.custom_id = addForm.custom_id
        }
        try {
          await createDevice(payload)
          successCount++
        } catch (e) {
          lastError = e.response?.data?.error || e.message
        }
      }
      if (successCount > 0) {
        ElMessage.success(`成功创建 ${successCount} 台设备`)
        addVisible.value = false
        loadDevices()
      } else {
        ElMessage.error('创建失败: ' + lastError)
      }
    } finally {
      addLoading.value = false
    }
  })
}

// 删除设备
const confirmDelete = (row) => {
  ElMessageBox.confirm(`确认删除设备 ${row.id}？`, '删除设备', {
    confirmButtonText: '删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteDevice(row.id)
      ElMessage.success('设备已删除')
      loadDevices()
    } catch (e) {
      ElMessage.error('删除失败: ' + (e.response?.data?.error || e.message))
    }
  }).catch(() => {})
}

onMounted(loadDevices)
</script>
