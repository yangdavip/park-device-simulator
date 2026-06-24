import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000
})

export const getStats = () => api.get('/stats')
export const getDevices = (params) => api.get('/devices', { params })
export const getDevice = (id) => api.get(`/devices/${id}`)
export const getDeviceData = (id) => api.get(`/devices/${id}/data`)
export const getScenarios = () => api.get('/scenarios')
export const activateScenario = (name) => api.post(`/scenarios/${name}/activate`)
export const getAlarms = () => api.get('/alarms')
export const ackAlarms = () => api.put('/alarms/0/ack')
export const getProtocolStatus = () => api.get('/protocols/status')
export const getProtocolInfo = () => api.get('/protocols/info')
export const createDevice = (data) => api.post('/devices', data)
export const deleteDevice = (id) => api.delete(`/devices/${id}`)

export default api
