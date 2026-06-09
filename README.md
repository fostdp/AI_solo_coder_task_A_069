# 数据中心制冷系统能效优化平台

超算中心制冷系统全栈监控与优化平台，覆盖 8 台离心式冷水机组、12 台冷却塔、80 台精密空调、20 台液冷 CDU 的实时数据采集、PUE 计算、冷量优化分配与分级告警。

## 架构

```
                          ┌─────────────────────────────────────────────────┐
                          │                 Frontend                         │
                          │  ┌──────────────┐  ┌──────────────────────────┐ │
                          │  │ cooling_3d.js │  │   pue_dashboard.js      │ │
                          │  │ Three.js 3D   │  │ Chart.js + D3 Sankey    │ │
                          │  └──────┬───────┘  └────────────┬─────────────┘ │
                          │         │    WebSocket + REST    │              │
                          └─────────┼────────────────────────┼──────────────┘
                                    │                        │
                          ┌─────────▼────────────────────────▼──────────────┐
                          │              Go Backend (main.go)               │
                          │                                                 │
                          │  ┌─────────────┐    chan DeviceDataEvent        │
                          │  │modbus_gateway├──────────────┬────────────────┐│
                          │  │连接池+数据采集│              │                ││
                          │  └──────┬──────┘              ▼                ││
                          │         │           ┌──────────────────┐        ││
                          │         │           │  pue_calculator  │        ││
                          │         │           │  PUE实时计算     │        ││
                          │         │           └────────┬─────────┘        ││
                          │         │                    │ chan PUEEvent    ││
                          │         │           ┌───────▼──────────┐        ││
                          │         │           │cooling_optimizer │        ││
                          │         │           │  冷量分配优化    │        ││
                          │         │           └──────────────────┘        ││
                          │         │                    │                   │
                          │  ┌──────▼───────────────────▼──────────────┐    │
                          │  │          alarm_notifier                  │    │
                          │  │   告警评估 + 钉钉推送 + 重试             │    │
                          │  └─────────────────────────────────────────┘    │
                          │                                                 │
                          │  /metrics  /debug/pprof/*  Gzip  CORS           │
                          └────────────────────┬────────────────────────────┘
                                               │
                          ┌────────────────────▼────────────────────────────┐
                          │              TimescaleDB                          │
                          │  device_data (超表,7天压缩,90天保留)             │
                          │  pue_records (超表,30天压缩,365天保留)           │
                          │  cooling_allocation (超表,30天压缩,180天保留)    │
                          │  device_data_5min (连续聚合物化视图)             │
                          └─────────────────────────────────────────────────┘
                                               ▲
                          ┌────────────────────┴────────────────────────────┐
                          │           Modbus TCP 模拟器                      │
                          │  120台设备 × 30s间隔 + HTTP控制API               │
                          │  端口 5020(Modbus) + 8081(HTTP控制)             │
                          └─────────────────────────────────────────────────┘
```

## 数据流

```
Modbus模拟器 ──TCP──► modbus_gateway ──channel──► pue_calculator ──channel──► cooling_optimizer
                          │                         │
                          └──channel─────────────────┴──► alarm_notifier ──► 钉钉
                                                                     │
                          WebSocket Hub ◄─────── 所有 channel ◄──────┘
                              │
                              ▼
                          前端实时更新
```

## 快速部署

### Docker Compose 一键启动

```bash
cd d:\AI_solo_coder_task_A\AI_solo_coder_task_A_069
docker-compose up -d
```

三容器自动编排：
1. **TimescaleDB** — 初始化数据库、创建超表和压缩策略
2. **Modbus模拟器** — 启动 120 台设备模拟，30 秒数据漂移
3. **Go后端** — 编译静态二进制，连接数据库和模拟器

访问 http://localhost:8080

### 端口映射

| 端口 | 服务 | 说明 |
|------|------|------|
| 8080 | Go后端 | HTTP + WebSocket + 静态文件 |
| 5432 | TimescaleDB | PostgreSQL |
| 5020 | Modbus模拟器 | Modbus TCP |
| 8081 | 模拟器控制API | HTTP 异常注入 |

### 本地开发

```bash
# 1. 启动 TimescaleDB
docker run -d --name dccooling-db \
  -p 5432:5432 \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=dccooling \
  -v $(pwd)/scripts/init_db.sql:/docker-entrypoint-initdb.d/init.sql \
  timescale/timescaledb:latest-pg15

# 2. 启动 Modbus 模拟器
python scripts/modbus_simulator.py --drift-interval 30

# 3. 启动 Go 后端
cd backend
go run ./cmd/

# 4. 浏览器访问
# http://localhost:8080
```

## 模拟器用法

### 命令行参数

```bash
python scripts/modbus_simulator.py [OPTIONS]

选项:
  --host HOST              Modbus监听地址 (默认: 0.0.0.0)
  --port PORT              Modbus监听端口 (默认: 5020)
  --control-port PORT      HTTP控制API端口 (默认: 8081)
  --drift-interval SECS    数据漂移间隔秒数 (默认: 30)
  --chillers N             冷水机组数量 (默认: 8)
  --cooling-towers N       冷却塔数量 (默认: 12)
  --precision-ac N         精密空调数量 (默认: 80)
  --cdu N                  液冷CDU数量 (默认: 20)
  --anomaly-rate RATE      随机COP异常概率 (默认: 0.005)
```

### 异常注入 API

模拟器在 `--control-port` 端口提供 HTTP 控制接口：

**查看状态**
```bash
curl http://localhost:8081/status
```

**注入COP异常 — 所有冷水机组COP降至2.5，持续5分钟**
```bash
curl -X POST http://localhost:8081/inject \
  -H 'Content-Type: application/json' \
  -d '{"device_type":"chiller","param":"cop","value":2.5,"duration":300}'
```

**注入异常 — 指定设备**
```bash
curl -X POST http://localhost:8081/inject \
  -H 'Content-Type: application/json' \
  -d '{"unit_id":1,"param":"cop","value":2.0,"duration":600}'
```

**注入异常 — 全部设备随机低COP**
```bash
curl -X POST http://localhost:8081/inject \
  -H 'Content-Type: application/json' \
  -d '{"param":"cop","duration":180}'
```

**清除所有异常**
```bash
curl -X POST http://localhost:8081/clear
```

### inject 参数说明

| 参数 | 类型 | 说明 |
|------|------|------|
| `device_type` | string | 设备类型: chiller/cooling_tower/precision_ac/cdu |
| `unit_id` | int | 指定设备单元ID (与device_type互斥) |
| `param` | string | 参数名: cop/supply_temp/return_temp/flow_rate/power/pressure/cooling_capacity |
| `value` | float | 强制设定值 (不填则自动生成异常值) |
| `duration` | int | 持续秒数 (不填则永久生效直到clear) |

## 监控

### Prometheus 指标

访问 http://localhost:8080/metrics

| 指标 | 类型 | 说明 |
|------|------|------|
| `dc_cooling_devices_collected_total` | Counter | 设备采集次数 (按类型/状态) |
| `dc_cooling_collection_duration_seconds` | Histogram | 采集耗时 |
| `dc_cooling_modbus_connections_active` | Gauge | 活跃Modbus连接数 |
| `dc_cooling_modbus_errors_total` | Counter | Modbus错误次数 |
| `dc_cooling_pue_value` | Gauge | 当前PUE值 |
| `dc_cooling_pue_calculation_duration_seconds` | Histogram | PUE计算耗时 |
| `dc_cooling_alerts_triggered_total` | Counter | 告警触发次数 (按级别/类型) |
| `dc_cooling_optimization_duration_seconds` | Histogram | 优化计算耗时 |
| `dc_cooling_websocket_clients` | Gauge | WebSocket连接数 |
| `dc_cooling_http_requests_total` | Counter | HTTP请求次数 |
| `dc_cooling_http_duration_seconds` | Histogram | HTTP请求耗时 |

### pprof 性能分析

```
go tool pprof http://localhost:8080/debug/pprof/profile    # CPU
go tool pprof http://localhost:8080/debug/pprof/heap       # 内存
go tool pprof http://localhost:8080/debug/pprof/goroutine  # 协程
go tool pprof http://localhost:8080/debug/pprof/trace      # 追踪
```

## 配置

Go后端通过 `config.json` 加载配置，支持环境变量覆盖：

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `CONFIG_PATH` | config.json | 配置文件路径 |
| `DB_CONN` | postgres://... | 数据库连接串 |
| `MODBUS_ADDR` | localhost:5020 | Modbus地址 |
| `DINGTALK_WEBHOOK` | 空 | 钉钉机器人Webhook |
| `IT_POWER` | 1700 | IT设备功率(kW) |
| `DISTRIBUTION_LOSS` | 250 | 配电损耗(kW) |
| `OTHER_INFRA_POWER` | 50 | 其他基础设施(kW) |
| `HTTP_PORT` | :8080 | HTTP监听端口 |

## TimescaleDB 数据策略

| 超表 | 压缩 | 压缩策略 | 保留策略 |
|------|------|---------|---------|
| `device_data` | segmentby=device_id, orderby=time DESC | 7天后自动压缩 | 90天后删除 |
| `pue_records` | orderby=time DESC | 30天后自动压缩 | 365天后删除 |
| `cooling_allocation` | segmentby=area, orderby=time DESC | 30天后自动压缩 | 180天后删除 |

连续聚合 `device_data_5min` 每5分钟自动刷新，提供5分钟粒度聚合数据供查询。

## 项目结构

```
├── backend/
│   ├── cmd/main.go                  # 主入口
│   ├── config.json                  # 配置文件
│   ├── Dockerfile                   # 多阶段构建
│   └── internal/
│       ├── config/                  # 配置加载
│       ├── modbus_gateway/          # Modbus连接池+数据采集
│       ├── pue_calculator/          # PUE实时计算
│       ├── cooling_optimizer/       # 冷量分配优化
│       ├── alarm_notifier/          # 告警评估+钉钉推送
│       ├── metrics/                 # Prometheus指标
│       ├── db/                      # 数据库模型和查询
│       ├── api/                     # REST API
│       └── ws/                      # WebSocket Hub
├── frontend/
│   ├── index.html                   # 主页面
│   └── js/
│       ├── cooling_3d.js            # Three.js 3D可视化组件
│       └── pue_dashboard.js         # 仪表盘组件
├── scripts/
│   ├── init_db.sql                  # TimescaleDB初始化
│   └── modbus_simulator.py          # Modbus模拟器
└── docker-compose.yml               # 容器编排
```
