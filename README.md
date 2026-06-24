# ScholarFlow Web

该项目是 ScholarFlow 的 Web 阅读前端 🌐。它是一个独立的 Go 子项目，通过调用 `scholarflow_server` 提供的 REST API，渲染论文列表页与论文阅读页。

- 首页展示已上传论文列表
- 论文页展示标题、作者、年份、DOI 等元数据
- 已生成的 paper card 会按背景、问题、方法、实现、结果等字段渲染
- 证据片段以 Tufte 风格边注形式展示，便于追溯结论来源
- 展示当前后端任务状态

## 📁 目录结构

```text
scholarflow_web/
  cmd/web/                 # Web 服务入口
  internal/apiclient/      # 后端 API 客户端
  internal/web/            # Handler、视图模型、模板渲染、静态资源
  internal/web/templates/  # HTML 模板
  internal/web/static/     # CSS 静态资源
  Dockerfile               # Web 服务镜像构建文件
  go.mod                   # 独立 Go module
```

## 🚀 运行与部署

`scholarflow_web` 依赖 `scholarflow_server` 的 HTTP API。启动 Web 前，请先确保后端 API 已经可用。

默认依赖关系：

- Web: `http://localhost:8090`
- API: `http://localhost:8080`

### 方式一：随整套 ScholarFlow 一起启动

这是最推荐的方式，适合本地联调整体系统。后端与 web 是两个独立的 compose 栈：先把后端拉起来，再单独启动 web。

1. 进入后端目录：

```bash
cd /home/onemotre/workspace/scholarflow/scholarflow-server
```

1. 启动基础依赖：

```bash
docker compose up -d postgres redis minio grobid
```

1. 启动 API 与 worker（数据库 schema 在启动时自动迁移）：

```bash
docker compose up -d --build api worker
```

1. 启动 web（独立 compose，经宿主机端口访问后端 API）：

```bash
cd /home/onemotre/workspace/scholarflow/scholarflow-web
docker compose up -d --build
```

1. 打开浏览器：

```text
http://localhost:8090
```

### 方式二：单独启动 `scholarflow_web`

```bash
cd /home/onemotre/workspace/scholarflow/scholarflow-web
go run ./cmd/web
```

启动后访问：

```text
http://localhost:8090
```

如果后端 API 地址不是默认值，可以显式指定：

```bash
cd /home/onemotre/workspace/scholarflow/scholarflow-web
SCHOLARFLOW_API_URL=http://localhost:8080 WEB_ADDR=:8090 go run ./cmd/web
```

### 方式三：使用 Docker 单独构建运行

```bash
cd /home/onemotre/workspace/scholarflow/scholarflow-web
docker build -t scholarflow-web .
docker run --rm -p 8090:8090 \
  -e SCHOLARFLOW_API_URL=http://host.docker.internal:8080 \
  scholarflow-web
```

说明：

- 容器内默认暴露端口 `8090`
- 如果 API 运行在宿主机，`SCHOLARFLOW_API_URL` 需要指向容器可访问的地址
- 在 Linux 环境下，`host.docker.internal` 是否可用取决于 Docker 配置；如果不可用，请改成宿主机实际可达地址

## ⚙️ 环境变量

`scholarflow_web` 当前只使用两个环境变量：

| 变量名 | 默认值 | 说明 |
| --- | --- | --- |
| `SCHOLARFLOW_API_URL` | `http://localhost:8080` | 后端 API 基地址 |
| `WEB_ADDR` | `:8090` | Web 服务监听地址 |

## 🧭 页面路由

- `GET /healthz`：健康检查，返回 `ok`
- `GET /`：论文列表页
- `GET /papers/{id}`：论文阅读页
- `GET /static/*`：静态资源

## 🖥️ 页面行为说明

### 首页 `/`

- 展示后端返回的论文列表
- 每条记录显示标题或原始上传文件名
- 显示论文当前状态与基础元数据
- 如果列表为空，会提示先通过 API 上传 PDF

### 阅读页 `/papers/{id}`

- 展示论文标题、作者、年份、DOI
- 如果后端已经生成 `card`，页面会按字段渲染阅读摘要
- 证据按 `claim_key` + `claim_index` 分组，以边注形式附着在对应字段或列表条目（如每条结果）旁边；相邻引用以逗号分隔，并标注页码 `[p.N]`
- 图表会按其归属的语义位置内联展示为边注（标签 + 页码 + 说明），而非单独的图表列表
- 若阅读尚未完成，会显示当前处理状态

## 📝 开发进展日志

- 初始化 `scholarflow_web` 独立 Go module
- 增加后端 API client，用于获取论文列表与论文详情
- 建立基础视图渲染结构与错误页
- 增加 Tufte 风格模板与静态样式资源
- 支持论文列表页与论文阅读页
- 支持将证据按 `claim_key` + `claim_index` 分组并渲染为带页码的边注（相邻引用逗号分隔）
- 支持图表按语义锚点内联展示、链接和卡片字段内容
- 增加 Web 服务入口 `cmd/web`
- 增加 Dockerfile
- 接入整套系统的 Compose 启动流程
- 在根项目文档中补充 Web 启动说明
