# Rclone-Backup-Web V2.0 - 分布式备份系统（v1.1更新）

## 📋 版本更新说明 (v1.1)

本版本在原有 v1.0 基础上，根据设计文档 v1.1 增加了以下重要特性：

### 🎨 新拟态(Neumorphism)设计风格
- 采用柔和、现代的视觉设计语言
- 元素具有"浮凸"和"凹陷"的3D效果
- 支持浅色/深色主题切换
- 完全响应式设计，移动端友好

### 🌐 完整的国际化(i18n)支持
- Web UI 支持中文/英文切换
- 所有界面文本均可翻译
- 用户偏好自动保存

### 📝 结构化日志系统
- 基于事件码的日志记录
- 支持中英双语日志消息
- 机器可读的 JSON 格式
- 保持 rclone 原生日志不翻译

## 🚀 快速开始

### 1. 环境准备

```bash
# 克隆代码
git clone https://github.com/your-repo/rclone-backup-web.git
cd rclone-backup-web/v2

# 初始化环境配置
make init-env
```

### 2. 部署中央节点(Hub)

```bash
# 编辑配置文件
vim docker/hub/.env

# 必须修改的配置项：
# - DB_PASSWORD: 数据库密码
# - JWT_SECRET: JWT密钥（使用 make gen-jwt-secret 生成）
# - ENCRYPTION_KEY: 加密密钥（使用 make gen-key 生成）

# 启动 Hub
make deploy-hub

# 查看日志
make logs-hub
```

访问系统：
- Web UI: http://localhost
- API: http://localhost:8080
- 默认账号: admin/admin

### 3. 注册并部署Agent节点

#### 方法一：使用部署脚本（推荐）

```bash
./deploy.sh
# 选择 2: Register Agent
# 按提示操作
```

#### 方法二：手动注册

```bash
# 1. 在Web UI创建注册令牌
# 2. 使用令牌注册Agent

curl -X POST http://hub-domain/api/v1/agent/register \
  -H "Content-Type: application/json" \
  -d '{
    "token": "your-registration-token",
    "name": "agent-name"
  }'

# 3. 保存返回的 agent_id 和 api_key
# 4. 配置Agent

cd docker/agent
vim .env
# 填入 HUB_URL, AGENT_ID, AGENT_API_KEY

# 5. 启动Agent
docker-compose up -d
```

## 🎨 新拟态设计特性

### UI组件库

系统提供了完整的新拟态组件：

```css
/* 使用示例 */
.neu-card        /* 凸起卡片 */
.neu-card-flat   /* 扁平卡片 */
.neu-card-inset  /* 凹陷卡片 */
.neu-button      /* 标准按钮 */
.neu-input       /* 输入框 */
.neu-select      /* 下拉选择 */
.neu-toggle      /* 开关 */
.neu-progress    /* 进度条 */
.neu-badge       /* 徽章 */
.neu-table       /* 表格 */
```

### 主题切换

```javascript
// 切换主题
const toggleTheme = () => {
  const currentTheme = document.documentElement.getAttribute('data-theme');
  const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
  document.documentElement.setAttribute('data-theme', newTheme);
  localStorage.setItem('theme', newTheme);
};
```

## 🌐 国际化使用

### 前端切换语言

```javascript
import { useTranslation } from 'react-i18next';

function Component() {
  const { t, i18n } = useTranslation();
  
  // 切换语言
  const changeLanguage = (lng) => {
    i18n.changeLanguage(lng);
  };
  
  // 使用翻译
  return <h1>{t('dashboard.title')}</h1>;
}
```

### 添加新语言

1. 创建语言文件：
```json
// src/i18n/locales/ja-JP.json
{
  "app": {
    "title": "分散バックアップシステム"
  }
}
```

2. 注册语言：
```javascript
// src/i18n/index.ts
import jaJP from './locales/ja-JP.json';

const resources = {
  'ja': { translation: jaJP }
};
```

## 📝 结构化日志系统

### 日志事件码

系统使用事件码记录日志，便于分析和国际化：

```go
// 使用示例
log.Info("TaskExecutionStarted", map[string]interface{}{
    "task_name": taskName,
    "execution_id": executionID,
})

// 输出 (JSON格式)
{
  "timestamp": "2025-10-17T10:30:00Z",
  "level": "INFO",
  "event_code": "TaskExecutionStarted",
  "message": "任务 Daily Backup 开始执行，执行ID：exec-123",
  "details": {
    "task_name": "Daily Backup",
    "execution_id": "exec-123"
  }
}
```

### 自定义日志消息

添加新的日志消息模板：

```json
// shared/logger/messages/zh.json
{
  "CustomEvent": "自定义事件发生：{description}"
}
```

使用：
```go
log.Info("CustomEvent", map[string]interface{}{
    "description": "具体描述"
})
```

## 📱 响应式设计

系统完全支持响应式布局：

- **桌面端** (>1024px): 完整侧边栏 + 主内容区
- **平板端** (768-1024px): 可折叠侧边栏
- **移动端** (<768px): 汉堡菜单 + 全屏内容

### 断点定义

```css
/* 响应式断点 */
@media (max-width: 768px) { /* 移动端 */ }
@media (min-width: 769px) and (max-width: 1024px) { /* 平板 */ }
@media (min-width: 1025px) { /* 桌面端 */ }
```

## 🔧 开发指南

### 本地开发环境

```bash
# 启动后端服务
cd v2/hub
go run .

# 启动前端开发服务器
cd v2/hub/web
npm install
npm run dev
```

### 构建生产版本

```bash
# 构建所有组件
make build

# 构建特定组件
make build-hub    # 构建Hub
make build-agent  # 构建Agent
make build-web    # 构建Web UI
```

### 测试

```bash
# 运行所有测试
make test

# 运行特定测试
make test-hub
make test-agent
```

## 📊 系统监控

### 实时日志查看

```bash
# Hub日志
docker logs -f rclone-hub-api

# Agent日志
docker logs -f rclone-agent

# 结构化日志分析
docker logs rclone-hub-api | jq 'select(.event_code=="TaskExecutionCompleted")'
```

### 性能指标

系统提供以下监控指标：
- Agent在线状态
- 任务执行成功率
- 平均执行时间
- 传输数据量
- 系统资源使用

## 🔐 安全最佳实践

1. **生产环境配置**
   - 必须启用 HTTPS
   - 使用强密码和密钥
   - 定期轮换 API 密钥

2. **网络安全**
   - 使用防火墙限制端口访问
   - 配置反向代理（Nginx/Caddy）
   - 启用速率限制

3. **数据安全**
   - 定期备份数据库
   - 加密敏感配置
   - 审计日志记录

## 📚 常见问题

### Q: 如何添加新的存储类型？

A: 在 Web UI 的"远程存储"页面，选择对应的存储类型并粘贴 rclone 配置。

### Q: Agent离线后任务还会执行吗？

A: 是的，Agent具有本地回退机制，会根据缓存的配置继续执行计划任务。

### Q: 如何查看任务的实时日志？

A: 在"执行历史"页面点击具体的执行记录，可以看到实时更新的日志。

### Q: 支持哪些 rclone 存储后端？

A: 支持所有 rclone 官方支持的存储后端，包括 S3、Google Drive、OneDrive、FTP 等。

## 📈 路线图

- [ ] 支持更多语言（日语、韩语等）
- [ ] 添加 Webhook 通知
- [ ] 实现任务依赖关系
- [ ] 支持增量备份
- [ ] 添加数据恢复功能
- [ ] 集成 Prometheus 监控
- [ ] 支持任务模板

## 🤝 贡献

欢迎贡献代码、报告问题或提出建议！

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

- [Rclone](https://rclone.org/) - 强大的云存储同步工具
- [Neumorphism.io](https://neumorphism.io/) - 新拟态设计灵感
- 所有贡献者和用户

---

**版本**: 2.0.0 (v1.1)
**更新日期**: 2025-10-17