# 智慧园区设备模拟器 (Park Device Simulator)

> 智慧园区 IoT 设备数据模拟平台，覆盖 10 大设备系统、49 种设备类型、303 台设备实例、203+ 数据点，支持 MQTT / HTTP REST / Modbus TCP / OPC UA 四种协议。数据基于物理约束和时间规律模型生成，非纯随机。

## 技术栈

- **后端**: Go 1.21+ / Gin / SQLite
- **前端**: Vue3 + Element Plus + ECharts + Pinia + Vue Router
- **协议**: MQTT (paho) / HTTP REST / Modbus TCP / OPC UA (gopcua)
- **架构**: 插件式设备注册 + 多协议适配器 + 场景引擎 + 告警引擎 + 调度引擎

## 快速开始

### 后端

```bash
# 编译
go build -o bin/park-device-simulator ./cmd/simulator/

# 启动（使用默认配置）
./bin/park-device-simulator

# 启动（指定配置目录和端口）
./bin/park-device-simulator -config configs -port 8090
```

### 前端开发

```bash
cd frontend
npm install --registry=https://registry.npmmirror.com
npm run dev    # Vite dev server :3000，自动代理 /api 到 :8090
npm run build  # 构建到 web/dist/
```

### 访问

| 入口 | 地址 |
|------|------|
| 前端开发 | http://localhost:3000 |
| 生产 Web UI | http://localhost:8090/web/ |
| API 状态 | http://localhost:8090/api/v1/stats |

### 协议说明

| 协议 | 端口 | 说明 |
|------|------|------|
| MQTT | 1883 | 需本地启动 broker（如 `brew install mosquitto && mosquitto`） |
| HTTP | 8080 | 需有 callback server 接收设备事件 |
| Modbus TCP | 502/503 | 内置内存模拟寄存器，无需外部服务 |
| OPC UA | 4840 | 内置 gopcua server，无需外部服务 |

## 10 大设备系统

| 系统 | 设备数 | 设备类型 |
|------|--------|----------|
| 楼宇自控 (BAS) | 52 | AHU、FCU、FAU、冷水机组、冷却塔、水泵、水箱、送排风机、热交换器 |
| 智能照明 | 40 | 回路控制器、照度传感器、灯具控制器、场景面板 |
| 安防监控 | 26 | 网络摄像机、球机、视频分析、红外对射、电子围栏 |
| 门禁管理 | 20 | 门禁控制器、人脸终端、访客机、闸机 |
| 消防报警 | 51 | 烟感、温感、手报、消防栓、喷淋泵、防火门 |
| 智能停车 | 59 | 车牌识别、地磁、超声波、引导屏、充电桩 |
| 能源管理 | 11 | 电力仪表、水表、燃气表、冷热量表、光伏逆变器、储能电池 |
| 环境监测 | 26 | 温湿度、PM2.5、CO2、噪声、气体、气象站 |
| 电梯监控 | 6 | 电梯控制器、扶梯控制器 |
| 广播发布 | 12 | 广播终端、紧急广播 |

**合计：303 台设备实例，49 种设备类型**

## 数据生成模型

数据基于物理约束和时间规律生成，非纯随机：

- **时间规律**: 工作日/节假日高峰低谷、入住率随小时变化、用电负载曲线
- **物理约束**: 温度惯性模型、三相电压平衡、功率因数与负载率相关、电池 SOC 库仑计数
- **环境模型**: 室外温度正弦曲线、CO2 与入住率正相关、PM2.5 季节性变化、噪声水平与人员活动相关

## 8 种场景模式

| 场景 | 类型 | 说明 |
|------|------|------|
| normal_workday | 定时 | 正常工作日（默认） |
| weekend | 定时 | 周末低负荷 |
| holiday | 定时 | 节假日 |
| summer_peak | 定时 | 夏季高温，冷负荷 ×1.3 |
| winter_peak | 定时 | 冬季严寒，室外基准温度 2°C |
| fire_emergency | 突发 | 消防报警，注入 12 条告警（烟感×5、温感×3、手报×1、消火栓×1、喷淋泵×1、紧急广播×1） |
| power_outage | 突发 | 停电事件，注入 2 条告警（市电中断、电梯停运） |
| intrusion | 突发 | 安防入侵，注入 8 条告警（红外对射×4、电子围栏×2、视频分析×1、门禁×1） |

## API 接口

| 路径 | 方法 | 说明 |
|------|------|------|
| /api/v1/stats | GET | 总览统计（设备总数、在线率、系统分布、当前场景） |
| /api/v1/devices | GET | 设备列表（支持 system/status 过滤，含 last_data） |
| /api/v1/devices/:id | GET | 设备详情 |
| /api/v1/devices/:id/data | GET | 设备最新数据 |
| /api/v1/scenarios | GET | 场景列表 |
| /api/v1/scenarios/:name/activate | POST | 激活场景 |
| /api/v1/alarms | GET | 告警列表（最近 1000 条） |
| /api/v1/alarms/:id/ack | PUT | 确认/清除告警 |
| /api/v1/protocols/status | GET | 协议适配器状态（MQTT/HTTP/Modbus/OPC UA） |

## 前端页面

| 页面 | 路由 | 功能 |
|------|------|------|
| 总览仪表盘 | / | 设备在线率、系统分布、告警统计、ECharts 可视化 |
| 设备管理 | /devices | 设备列表、系统/状态筛选、关键字搜索、最新数据展示、设备详情 |
| 告警中心 | /alarms | 告警列表、等级筛选、确认清除 |
| 场景模式 | /scenarios | 8 种场景切换、当前场景状态 |
| 协议状态 | /protocols | 四种协议运行状态、连接信息 |

## 项目结构

```
park-device-simulator/
├── cmd/simulator/           # 主程序入口
├── configs/                 # YAML 配置文件
│   ├── park.yaml            # 园区/楼宇/设备配置
│   ├── protocols.yaml       # 协议配置
│   └── scenarios.yaml       # 场景配置
├── internal/
│   ├── api/                 # HTTP API 路由（Gin）
│   ├── config/              # 配置加载
│   ├── device/              # 设备接口 + 10 个系统实现
│   │   ├── base.go          # Device 接口 + BaseDevice
│   │   ├── bas/             # 楼宇自控（9 种）
│   │   ├── lighting/        # 智能照明（4 种）
│   │   ├── security/        # 安防监控（5 种）
│   │   ├── access/          # 门禁管理（4 种）
│   │   ├── fire/            # 消防报警（6 种）
│   │   ├── parking/         # 智能停车（5 种）
│   │   ├── energy/          # 能源管理（6 种）
│   │   ├── environment/     # 环境监测（6 种）
│   │   ├── elevator/        # 电梯监控（2 种）
│   │   └── broadcast/       # 广播发布（2 种）
│   ├── engine/              # 核心引擎
│   │   ├── scenario.go      # 场景引擎（含突发场景告警注入）
│   │   ├── alarm.go         # 告警引擎（规则匹配 + 持续时间判定）
│   │   ├── scheduler.go     # 调度引擎（设备定时上报）
│   │   └── utils.go         # 数据生成工具函数
│   ├── protocol/            # 协议适配器
│   │   ├── mqtt/            # MQTT 适配器（paho）
│   │   ├── http/            # HTTP REST 适配器
│   │   ├── modbus/          # Modbus TCP 适配器（内存寄存器）
│   │   └── opcua/           # OPC UA 适配器（gopcua server）
│   └── types/               # 公共类型定义
├── frontend/                # Vue3 前端源码
│   ├── src/
│   │   ├── api/index.js     # API 封装（axios）
│   │   ├── assets/main.css  # 全局样式
│   │   ├── router/index.js  # Vue Router
│   │   ├── views/           # 5 个页面组件
│   │   │   ├── Overview.vue     # 总览仪表盘
│   │   │   ├── Devices.vue      # 设备管理
│   │   │   ├── Alarms.vue       # 告警中心
│   │   │   ├── Scenarios.vue    # 场景模式
│   │   │   └── Protocols.vue    # 协议状态
│   │   └── App.vue          # 根组件
│   ├── vite.config.js       # Vite 配置（dev/build 条件 base）
│   └── package.json
├── web/dist/                # 前端构建输出（gitignored）
├── docs/                    # 需求设计方案文档
├── go.mod
└── README.md
```

## 代码统计

| 模块 | 行数 |
|------|------|
| Go 后端 | ~5,500 行 |
| Vue 前端 | ~780 行 |
| YAML 配置 | ~260 行 |

## 仓库

- **Gitee**: https://gitee.com/yang_davip/park-device-simulator
- **GitHub**: https://github.com/yangdavip/park-device-simulator

## License

MIT
