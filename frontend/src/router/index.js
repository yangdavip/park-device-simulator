import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', redirect: '/overview' },
  { path: '/overview', name: 'overview', component: () => import('../views/Overview.vue'), meta: { title: '总览仪表盘', icon: 'Odometer' } },
  { path: '/devices', name: 'devices', component: () => import('../views/Devices.vue'), meta: { title: '设备管理', icon: 'Cellphone' } },
  { path: '/alarms', name: 'alarms', component: () => import('../views/Alarms.vue'), meta: { title: '告警中心', icon: 'Warning' } },
  { path: '/scenarios', name: 'scenarios', component: () => import('../views/Scenarios.vue'), meta: { title: '场景模式', icon: 'Film' } },
  { path: '/protocols', name: 'protocols', component: () => import('../views/Protocols.vue'), meta: { title: '协议状态', icon: 'Connection' } },
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
