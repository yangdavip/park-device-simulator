<template>
  <div class="app-layout">
    <el-aside :width="collapsed ? '64px' : '220px'" class="app-aside">
      <div style="padding: 16px; text-align: center; border-bottom: 1px solid #1e2d5a;">
        <span v-if="!collapsed" style="color: #4fc3f7; font-size: 15px; font-weight: 600;">🏢 园区设备模拟器</span>
        <span v-else style="color: #4fc3f7; font-size: 18px;">🏢</span>
      </div>
      <el-menu
        :default-active="$route.path"
        :collapse="collapsed"
        router
        background-color="transparent"
        text-color="#8892b0"
        active-text-color="#4fc3f7"
      >
        <el-menu-item v-for="r in menuRoutes" :key="r.path" :index="r.path">
          <el-icon><component :is="r.meta.icon" /></el-icon>
          <template #title>{{ r.meta.title }}</template>
        </el-menu-item>
      </el-menu>
      <div style="position: absolute; bottom: 16px; width: 100%; text-align: center;">
        <el-icon @click="collapsed = !collapsed" style="cursor: pointer; color: #8892b0; font-size: 18px;">
          <Fold v-if="!collapsed" />
          <Expand v-else />
        </el-icon>
      </div>
    </el-aside>

    <el-container>
      <el-header class="app-header">
        <span class="title">{{ $route.meta.title || '智慧园区设备监控平台' }}</span>
        <div style="flex: 1"></div>
        <el-tag :type="online ? 'success' : 'danger'" effect="dark" size="small">
          {{ online ? '● 运行中' : '● 已停止' }}
        </el-tag>
        <span style="margin-left: 16px; color: #8892b0; font-size: 13px;">{{ currentTime }}</span>
      </el-header>

      <el-main class="app-main">
        <router-view />
      </el-main>
    </el-container>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { getStats } from './api'
import dayjs from 'dayjs'

const router = useRouter()
const collapsed = ref(false)
const online = ref(true)
const currentTime = ref('')
const stats = ref(null)

let timer = null
let pollTimer = null

const menuRoutes = computed(() =>
  router.options.routes.filter(r => r.meta && r.meta.title)
)

const updateTime = () => {
  currentTime.value = dayjs().format('YYYY-MM-DD HH:mm:ss')
}

const pollStats = async () => {
  try {
    const { data } = await getStats()
    stats.value = data
    online.value = data.online_devices > 0
  } catch {
    online.value = false
  }
}

onMounted(() => {
  updateTime()
  timer = setInterval(updateTime, 1000)
  pollStats()
  pollTimer = setInterval(pollStats, 10000)
})

onUnmounted(() => {
  clearInterval(timer)
  clearInterval(pollTimer)
})
</script>

<style scoped>
.app-layout { height: 100vh; }
.app-aside { position: relative; border-right: 1px solid #1e2d5a; }
</style>
