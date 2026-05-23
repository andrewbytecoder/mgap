# MGAP - monitor go application profile

[English](README.md) | [简体中文](README_zh.md)

**mgap**（monitor go application profile）是一款基于 [Wails](https://wails.io) 的轻量级桌面 pprof 监控工具。它实时获取、解析并可视化 Go 运行时性能分析数据，无需在浏览器和本地 Web 服务之间来回切换。

![img.png](screenshots/endpoint-detection.png)

![img.png](screenshots/mgap.gif)

---

## 功能特性

- **实时 Profile 采样** — 持续采样 heap、CPU、allocs、goroutine、block、mutex、threadcreate 等 profile 数据。
- **自动 Endpoint 检测** — 只需输入端口号（如 `6060`）或部分 URL，应用即可自动发现 `/debug/pprof` 端点。
- **实时图表** — 基于 ECharts 的 Top 函数时间线图表，支持时间范围选择和 Top-N 筛选。
- **火焰图视图** — 交互式火焰图，快速定位热点函数。
- **导入 / 导出** — 导入已有的 profile 文件，或将捕获的快照导出以供离线分析。
- **纯文本视图** — 类似浏览器的原始 profile 文本输出，方便查看具体数值。
- **Mock 数据模式** — 内置模拟数据，无需运行中的 Go 进程即可测试 UI。
- **无边框桌面外壳** — 自定义标题栏，支持拖拽移动、边缘吸附最大化 / 左右半屏、最小化和关闭。
- **跨平台** — 通过 Wails 支持 Windows、macOS 和 Linux。

---

## 技术栈

| 层级 | 技术 |
|---|---|
| 后端 | Go 1.23 |
| 桌面壳 | Wails v2 |
| 前端 | Vue 3 + TypeScript |
| UI 组件 | Vuetify 3 |
| 图表 | ECharts + vue-echarts |
| 协议缓冲区 | google.golang.org/protobuf |

---

## 快速开始

### 前置条件

- [Go](https://go.dev/dl/) 1.23+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation) v2.12+
- [Node.js](https://nodejs.org/) 18+

### 开发运行

```bash
wails dev
```

这会同时启动 Go 后端和 Vite 前端开发服务器，并支持热重载。

### 构建打包

```bash
wails build
```

打包后的可执行文件位于：

- **Windows**：`build/bin/mgap.exe`
- **macOS**：`build/bin/mgap.app`
- **Linux**：`build/bin/mgap`

---

## 使用方法

### 1. 在 Go 应用中暴露 pprof 端点

```go
package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	log.Println(http.ListenAndServe("localhost:6060", nil))
}
```

### 2. 打开 mgap 并连接

在 **Endpoint** 输入框中输入以下内容之一：

```text
6060
localhost:6060
http://localhost:6060/debug/pprof
```

应用会自动规范化输入并探测可用的 profile。

### 3. 开始采样

点击 **Start** 开始周期性采样。可以单独开关各个指标，在 **Flat** 和 **Cumulative** 视图之间切换，并调整 **Top N** 筛选值。

---

## 快捷键与窗口行为

- **拖拽标题栏** — 移动窗口。
- **拖拽到屏幕顶部** — 最大化。
- **拖拽到屏幕左 / 右边缘** — 吸附为左 / 右半屏。
- **双击标题栏** — 切换最大化状态。

---

## 注意事项

- Profile 历史数据保存在内存中，应用重启后会重置。
- 保留样本数过大时，图表渲染可能会变重。
- 本工具面向**本地开发和快速排查**，不能替代 Prometheus / Grafana 等生产级监控系统。

---

## License

MIT
