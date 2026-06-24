<template>
  <div>
    <!-- 协议运行状态 -->
    <el-divider content-position="left">协议运行状态</el-divider>
    <el-row :gutter="16">
      <el-col :span="6" v-for="(status, proto) in protocols" :key="proto">
        <el-card shadow="hover" style="margin-bottom: 16px;">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <span style="font-weight: 600; color: #4fc3f7;">{{ proto.toUpperCase() }}</span>
              <el-tag :type="isRunning(proto, status) ? 'success' : 'danger'" size="small">
                {{ isRunning(proto, status) ? '运行中' : '未连接' }}
              </el-tag>
            </div>
          </template>
          <el-descriptions :column="1" size="small" border>
            <el-descriptions-item v-for="(v, k) in status" :key="k" :label="k">
              <span v-if="typeof v === 'boolean'" :style="{color: v ? '#67c23a' : '#909399'}">
                {{ v ? '是' : '否' }}
              </span>
              <span v-else-if="typeof v === 'object'">{{ JSON.stringify(v) }}</span>
              <span v-else>{{ v }}</span>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <!-- 对接信息 -->
    <el-divider content-position="left">外部平台对接信息</el-divider>
    <el-row :gutter="16">
      <el-col :span="12" v-for="info in protocolInfoList" :key="info.title">
        <el-card shadow="hover" style="margin-bottom: 16px;">
          <template #header>
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <span style="font-weight: 600;">{{ info.title }}</span>
              <el-tag size="small" :type="info.tagType">{{ info.direction }}</el-tag>
            </div>
          </template>
          <el-descriptions :column="1" size="small" border>
            <el-descriptions-item v-for="item in info.items" :key="item.label" :label="item.label">
              <el-link v-if="item.copy" type="primary" @click="copyText(item.value)" style="font-family: monospace;">
                {{ item.value }} 📋
              </el-link>
              <span v-else-if="item.code" style="font-family: monospace; background: #f5f7fa; padding: 2px 6px; border-radius: 3px; font-size: 12px;">
                {{ item.value }}
              </span>
              <span v-else>{{ item.value }}</span>
            </el-descriptions-item>
          </el-descriptions>
          <div v-if="info.note" style="margin-top: 8px; color: #909399; font-size: 12px;">
            💡 {{ info.note }}
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="hover" v-if="Object.keys(protocols).length === 0 && !loading">
      <el-empty description="暂无协议状态数据，请确认模拟器已启动" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { getProtocolStatus, getProtocolInfo } from '../api'

const protocols = ref({})
const protocolInfo = ref({})
const loading = ref(false)

const isRunning = (proto, status) => {
  if (proto === 'mqtt') return status.connected > 0 || !status.broker_down
  if (proto === 'http') return status.connected === true
  if (proto === 'opcua') return status.running === true
  if (proto === 'modbus') return status.servers && status.servers.length > 0
  return false
}

const protocolInfoList = computed(() => {
  const list = []
  const info = protocolInfo.value

  if (info.mqtt) {
    list.push({
      title: 'MQTT',
      direction: '模拟器 → Broker → 平台',
      tagType: 'primary',
      note: info.mqtt.note,
      items: [
        { label: 'Broker 地址', value: `${info.mqtt.broker_host}:${info.mqtt.broker_port}`, copy: true },
        { label: '数据 Topic', value: info.mqtt.topic_pattern, code: true, copy: true },
        { label: '告警 Topic', value: info.mqtt.alarm_topic, code: true, copy: true },
        { label: 'QoS', value: info.mqtt.qos },
        { label: 'Payload 格式', value: info.mqtt.payload_format.toUpperCase() },
      ],
    })
  }

  if (info.http) {
    list.push({
      title: 'HTTP REST',
      direction: '模拟器 → 平台',
      tagType: 'success',
      note: info.http.note,
      items: [
        { label: 'Callback URL', value: info.http.callback_url, copy: true },
        { label: '请求方法', value: info.http.method },
        { label: '请求路径', value: info.http.path_pattern, code: true, copy: true },
        { label: 'Payload 格式', value: info.http.payload_format.toUpperCase() },
      ],
    })
  }

  if (info.modbus) {
    const servers = info.modbus.servers || []
    const serverDesc = servers.map(s => `${s.name}(:${s.port}, slave ${s.slave_ids.join(',')})`).join('; ')
    list.push({
      title: 'Modbus TCP',
      direction: '平台 → 模拟器',
      tagType: 'warning',
      note: info.modbus.note,
      items: [
        { label: '从站服务', value: serverDesc },
        { label: '寄存器类型', value: info.modbus.register_type },
        { label: '数据格式', value: info.modbus.data_format },
        { label: '寄存器映射', value: '详见 API /protocols/info', code: false },
      ],
    })
  }

  if (info.opcua) {
    list.push({
      title: 'OPC UA',
      direction: '平台 → 模拟器',
      tagType: 'warning',
      note: info.opcua.note,
      items: [
        { label: 'Endpoint', value: info.opcua.endpoint, copy: true },
        { label: '命名空间', value: info.opcua.namespace },
        { label: '安全策略', value: info.opcua.security },
        { label: '节点格式', value: info.opcua.node_pattern, code: true, copy: true },
        { label: '值类型', value: info.opcua.value_type },
      ],
    })
  }

  return list
})

const copyText = (text) => {
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success('已复制: ' + text)
  }).catch(() => {
    ElMessage.warning('复制失败')
  })
}

const load = async () => {
  loading.value = true
  try {
    const [statusRes, infoRes] = await Promise.all([
      getProtocolStatus(),
      getProtocolInfo()
    ])
    protocols.value = statusRes.data || {}
    protocolInfo.value = infoRes.data || {}
  } catch (e) {
    console.error('load protocols error', e)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>
