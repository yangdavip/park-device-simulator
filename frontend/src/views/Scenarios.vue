<template>
  <div>
    <el-card shadow="hover">
      <template #header>场景模式管理</template>
      <el-row :gutter="16">
        <el-col :span="6" v-for="s in scenarios" :key="s.name">
          <el-card
            shadow="hover"
            :class="{ 'scenario-active': s.active }"
            style="cursor: pointer; margin-bottom: 16px;"
            @click="activate(s.name)"
          >
            <div style="display: flex; justify-content: space-between; align-items: center;">
              <div>
                <div style="font-size: 16px; font-weight: 600; color: #e0e0e0;">{{ s.description || s.name }}</div>
                <div style="font-size: 12px; color: #8892b0; margin-top: 4px;">{{ s.name }}</div>
                <el-tag size="small" :type="s.type === 'schedule' ? 'info' : 'danger'" style="margin-top: 8px;">
                  {{ s.type === 'schedule' ? '定时' : '瞬时' }}
                </el-tag>
              </div>
              <div>
                <el-icon v-if="s.active" style="color: #66bb6a; font-size: 24px;"><CircleCheckFilled /></el-icon>
                <el-icon v-else style="color: #8892b0; font-size: 24px;"><CircleClose /></el-icon>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="hover" style="margin-top: 16px;" v-if="currentScenario">
      <template #header>当前场景参数</template>
      <el-descriptions :column="2" border>
        <el-descriptions-item label="场景名称">{{ currentScenario.name }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ currentScenario.type }}</el-descriptions-item>
        <el-descriptions-item label="描述" :span="2">{{ currentScenario.description }}</el-descriptions-item>
        <el-descriptions-item v-for="(v, k) in currentScenario.overrides" :key="k" :label="k">{{ v }}</el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { getScenarios, activateScenario } from '../api'
import { ElMessage } from 'element-plus'

const scenarios = ref([])

const currentScenario = computed(() => scenarios.value.find(s => s.active))

const activate = async (name) => {
  try {
    await activateScenario(name)
    ElMessage.success(`场景已切换: ${name}`)
    await loadScenarios()
  } catch (e) {
    ElMessage.error('场景切换失败')
  }
}

const loadScenarios = async () => {
  try {
    const { data } = await getScenarios()
    scenarios.value = data.scenarios || []
  } catch (e) {
    console.error('loadScenarios error', e)
  }
}

onMounted(loadScenarios)
</script>

<style scoped>
.scenario-active {
  border-color: #66bb6a !important;
  box-shadow: 0 0 12px rgba(102, 187, 106, 0.3) !important;
}
</style>
