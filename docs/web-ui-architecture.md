# Web UI 架构说明

## 🏗️ 为什么需要 Nginx？

这是一个常见的疑问：既然我们已经有了Hub API服务，为什么Web UI还需要Nginx？

### 架构分层

```
┌─────────────────────────────────────────┐
│           用户浏览器                      │
└────────────┬────────────────┬────────────┘
             │                │
         (静态文件)         (API请求)
             │                │
             ▼                ▼
    ┌──────────────┐   ┌──────────────┐
    │  Nginx       │   │  Hub API     │
    │  (Port 3000) │   │  (Port 8080) │
    ├──────────────┤   ├──────────────┤
    │ 静态文件服务  │   │  业务逻辑    │
    │ index.html   │   │  数据库操作   │
    │ JS/CSS文件   │   │  Agent管理   │
    └──────────────┘   └──────────────┘
```

### 职责分离

1. **Hub API (Go服务)**
   - 处理业务逻辑
   - 管理数据库
   - 与Agent通信
   - 提供RESTful API
   - 处理WebSocket/SSE

2. **Web UI (Nginx + 静态文件)**
   - 提供React/Vue应用的静态文件
   - 处理SPA路由（返回index.html）
   - 静态资源缓存
   - Gzip压缩
   - 安全头部

## 📦 两阶段构建

Web UI的Dockerfile采用多阶段构建模式：

### 第一阶段：构建（Builder）

```dockerfile
FROM node:20-alpine AS builder
# 包含完整的Node.js环境
# 安装所有依赖（包括devDependencies）
# 执行构建命令
# 生成dist目录
```

**为什么需要所有依赖？**
- `vite`: 构建工具（devDependency）
- `@vitejs/plugin-react`: Vite插件（devDependency）
- `typescript`: 类型检查（devDependency）
- `eslint`: 代码检查（devDependency）

这些工具只在构建时需要，运行时不需要。

### 第二阶段：运行（Runtime）

```dockerfile
FROM nginx:1.25-alpine
# 精简的Alpine Linux + Nginx
# 仅复制构建产物（dist目录）
# 不包含Node.js或任何构建工具
# 镜像大小：~25MB vs ~300MB
```

## 🚀 为什么这样设计？

### 1. 性能优化
- **Nginx擅长提供静态文件**：比Node.js快10倍以上
- **零CPU开销**：静态文件不需要服务端计算
- **高并发**：Nginx可以处理数万并发连接

### 2. 安全性
- **最小攻击面**：运行时镜像不包含构建工具
- **权限分离**：Nginx运行在非root用户
- **无Node.js漏洞**：运行时不包含Node.js

### 3. 资源效率
- **镜像体积**：25MB vs 300MB
- **内存占用**：10MB vs 100MB+
- **启动速度**：<1秒 vs 5-10秒

## 📝 nginx.conf 的作用

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    
    # SPA路由处理
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
    
    # Gzip压缩
    gzip on;
    gzip_types text/plain text/css application/json application/javascript;
}
```

### 关键配置说明

1. **SPA路由兼容**
   ```nginx
   try_files $uri $uri/ /index.html;
   ```
   - 用户访问 `/tasks` → 返回 `index.html`
   - Vue Router接管路由 → 显示Tasks页面

2. **缓存策略**
   - 静态资源：1年缓存
   - HTML文件：不缓存（总是最新）

3. **压缩**
   - 自动压缩文本文件
   - 减少70%网络传输

## 🔧 常见问题

### Q: 为什么不用Node.js直接serve静态文件？

A: 可以，但不推荐：
- Node.js不擅长处理静态文件
- 占用更多内存和CPU
- 需要额外的库（express-static等）
- 性能差异明显（特别是高并发时）

### Q: 为什么不让Hub API同时serve静态文件？

A: 职责分离原则：
- API专注于业务逻辑
- 静态服务器专注于文件分发
- 独立扩展（可以单独扩容Web UI）
- 独立部署（前后端分离）

### Q: npm ci --only=production 错误的原因？

A: 构建时序问题：
```
npm ci --only=production  → 只安装运行时依赖
npm run build            → 需要vite（开发依赖）
                         → 找不到vite → 失败
```

正确做法：
```
npm ci                   → 安装所有依赖
npm run build            → vite可用 → 成功
最终镜像                  → 只包含dist，不包含node_modules
```

## 📊 对比

| 方案 | 镜像大小 | 内存占用 | 并发能力 | 复杂度 |
|-----|---------|---------|---------|-------|
| Node.js serve | ~300MB | ~100MB | 低 | 简单 |
| Nginx | ~25MB | ~10MB | 极高 | 中等 |
| Hub API serve | 0 | 0 | 中等 | 复杂 |

## 🎯 最佳实践

1. **开发环境**：使用Vite dev server（热更新）
2. **生产环境**：使用Nginx（性能）
3. **构建阶段**：安装所有依赖
4. **运行阶段**：只包含必要文件

这种架构是业界标准，被Facebook、Google、Netflix等公司广泛采用。