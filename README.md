# 智慧园区设备模拟器 (Park Device Simulator)

> 智慧园区 IoT 设备数据模拟平台，覆盖 10 大设备系统、49 种设备类型、303+ 数据点，支持 MQTT / HTTP REST / Modbus TCP / OPC UA 四种协议。

## 技术栈

- **后端**: Go 1.21+ / Gin / SQLite
- **前端**: Vue3 + Element Plus + ECharts + Pinia + Vue Router
- **协议**: MQTT (paho) / HTTP REST / Modbus TCP / OPC UA (Phase 4)
- **架构**: 插件式设备注册 + 多协议适配器 + 场景引擎 + 告警引擎

## 快速开始

### 后端

```bash
# 编译
go build -o bin/park-device-simulator ./cmd/simulator/

# 启动（使用默认配置）
./bin/park-device-simulator

# 启动（指定配置目录）
./bin/park-device-simulator -config configs -port 8090
```

### 前端开发

```bash
cd frontend
npm install --registry=https://registry.npmmirror.com
npm run dev   # Vite dev server :3000，自动代理 /api 到 :8090
npm run build # 构建到 web/dist/
```

访问 Web UI: http://localhost:8090/web/
API 状态: http://localhost:8090/api/v1/stats

## 10 大设备系统

| 系统 | 设备数 | 设备类型 |
|------|--------|----------|
| 楼宇自控 (BAS) | 9 | AHU, FCU, FAU, 冷水机组, 冷却塔, 水泵, 水箱, 送排风机, 热交换器 |
| 智能照明 | 4 | 回路控制器, 照度传感器, 灯具控制器, 场景面板 |
| 安防监控 | 5 | 网络摄像机, 球机, 视频分析, 红外对射, 电子围栏 |
| 门禁管理 | 4 | 门禁控制器, 人脸终端, 访客机, 闸机 |
| 消防报警 | 6 | 烟感, 温感, 手报, 消防栓, 喷淋泵, 防火门 |
| 智能停车 | 5 | 车牌识别, 地磁, 超声波, 引导屏, 充电桩 |
| 能源管理 | 6 | 电力仪表, 水表, 燃气表, 冷热量表, 光伏逆变器, 储能电池 |
| 环境监测 | 6 | 温湿度, PM2.5, CO2, 噪声, 气体, 气象站 |
| 电梯监控 | 2 | 电梯控制器, 扶梯控制器 |
| 广播发布 | 2 | 广播终端, 紧急广播 |

## 数据生成模型

数据基于物理约束和时间规律生成，非纯随机：

- **时间规律**: 工作日/节假日高峰低谷、入住率随小时变化
- **物理约束**: 温度惯性、三相电压平衡、功率因数与负载率相关
- **8 种场景**: 正常工作日、周末、节假日、夏冬高峰、消防突发、停电、入侵

## API

| 路径 | 方法 | 说明 |
|------|------|------|
| /api/v1/stats | GET | 总览统计 |
| /api/v1/devices | GET | 设备列表（支持 system/status 过滤） |
| /api/v1/devices/:id | GET | 设备详情 |
| /api/v1/devices/:id/data | GET | 设备最新数据 |
| /api/v1/scenarios | GET | 场景列表 |
| /api/v1/scenarios/:name/activate | POST | 激活场景 |
| /api/v1/alarms | GET | 告警列表 |
| /api/v1/alarms/:id/ack | PUT | 确认告警 |
| /api/v1/protocols/status | GET | 协议适配器状态 |

## 项目结构

```
park-device-simulator/
├── cmd/simulator/          # 主程序入口
├── configs/                # YAML 配置文件
│   ├── park.yaml           # 园区/楼宇/设备配置
│   ├── protocols.yaml      # 协议配置
│   └── scenarios.yaml      # 场景配置
├── internal/
│   ├── api/                # HTTP API 路由
│   ├── config/             # 配置加载
│   ├── device/             # 设备接口 + 10 个系统实现
│   │   ├── bas/            # 楼宇自控
│   │   ├── lighting/       # 智能照明
│   │   ├── security/       # 安防监控
│   │   ├── access/         # 门禁管理
│   │   ├── fire/           # 消防报警
│   │   ├── parking/        # 智能停车
│   │   ├── energy/         # 能源管理
│   │   ├── environment/    # 环境监测
│   │   ├── elevator/       # 电梯监控
│   │   └── broadcast/      # 广播发布
│   ├── engine/             # 数据生成 + 场景 + 告警 + 调度
│   ├── protocol/           # 协议适配器
│   │   ├── mqtt/           # MQTT 适配器
│   │   ├── http/           # HTTP 适配器
│   │   ├── modbus/         # Modbus TCP 适配器
│   │   └── opcua/          # OPC UA (Phase 4)
│   └── types/              # 公共类型
├── frontend/                # Vue3 前端源码
│   ├── src/
│   │   ├── api/            # API 封装
│   │   ├── assets/         # 全局样式
│   │   ├── components/     # 公共组件
│   │   ├── router/         # Vue Router
│   │   ├── views/          # 5 个页面组件
│   │   └── App.vue         # 根组件
│   ├── vite.config.js
│   └── package.json
├── web/                    # 前端构建输出 + 静态文件
│   ├── dist/              # vite build 输出
│   └── index.html         # 旧版纯 HTML（备用）
└── docs/                   # 需求设计方案文档
```

## License

MIT
