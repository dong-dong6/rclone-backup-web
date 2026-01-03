# V2 Web UI Style Migration

## 概述

本文档记录了将 V2 Web UI 的设计风格统一至与 templates/ 目录中原始风格一致的所有修改。

## 设计风格对比

### Templates 原始风格特点
- **主色调**: 黑白渐变 (#000000 到 #434343)
- **卡片效果**: 玻璃态毛玻璃效果
- **字体**: Inter 字体系列
- **渐变**: 多处使用渐变效果
- **圆角**: 12px 大圆角, 8px 小圆角
- **阴影**: 多层次阴影系统

### V2 原始风格
- **主色调**: Ant Design 蓝色 (#1890ff)
- **背景**: 紫色渐变登录页
- **卡片**: 简单白色卡片
- **字体**: 系统默认字体

## 修改的文件清单

### 1. `/v2/hub/web/src/index.css`
**修改内容**:
- 添加 Inter 字体导入
- 添加完整的 CSS 变量系统 (与 templates/base.html 一致)
- 更新 body 背景为白色渐变
- 更新卡片样式为玻璃态效果
- 添加 hover 动画效果

**关键变量**:
```css
--primary-gradient: linear-gradient(135deg, #000000 0%, #434343 100%);
--border-radius: 12px;
--border-radius-sm: 8px;
--shadow-light: 0 4px 20px rgba(0, 0, 0, 0.08);
```

### 2. `/v2/hub/web/src/App.css`
**修改内容**:
- 更新布局背景为白色渐变
- 侧边栏使用深色渐变背景
- 顶部导航栏使用浅色渐变
- logo 区域黑色渐变背景
- 自定义滚动条样式
- 覆盖 Ant Design 组件默认样式
  - Card: 玻璃态效果
  - Button: 黑色渐变主按钮
  - Table: 渐变表头
  - Menu: 圆角菜单项

### 3. `/v2/hub/web/src/pages/Login.css`
**修改内容**:
- 登录页背景从紫色渐变改为白色渐变
- 登录卡片玻璃态效果
- 卡片头部黑色渐变
- 自定义表单输入框样式
- 按钮黑色渐变效果
- 添加 hover 动画

### 4. `/v2/hub/web/src/pages/Dashboard.css` (新建)
**新增内容**:
- 统计卡片样式 (包含顶部彩色横条)
- 图表容器样式
- 表格样式覆盖
- 动画效果 (fadeInUp, pulse)
- 状态标签样式
- 进度条样式

### 5. `/v2/hub/web/src/pages/Dashboard.tsx`
**修改内容**:
- 导入 Dashboard.css
- 添加 className="dashboard-card" 到所有卡片
- 添加 className="dashboard-stats-card" 到统计卡片
- 为不同状态的卡片添加 success/warning/danger 类名
- 更新图表颜色为新配色方案
- 添加 fade-in-up 动画类
- 优化文本样式和字重

### 6. `/v2/hub/web/src/main.tsx`
**修改内容**:
- 将 App 导入从 App.neu 改为 App
- 移除旧的样式表导入
- 添加完整的 Ant Design 主题配置
  - colorPrimary: #000000 (黑色)
  - colorSuccess: #28a745 (绿色)
  - colorWarning: #ffc107 (黄色)
  - colorError: #dc3545 (红色)
  - 字体: Inter
  - 圆角: 8px / 12px
  - 组件级别的样式覆盖

## 颜色系统

### 主色调
- **Primary**: `#000000` → `#434343` (黑色渐变)
- **Success**: `#28a745` → `#20c997` (绿色渐变)
- **Warning**: `#ffc107` → `#fd7e14` (橙色渐变)
- **Danger**: `#dc3545` → `#e83e8c` (红色渐变)

### 文本颜色
- **Primary Text**: `#212529`
- **Secondary Text**: `#6c757d`
- **Muted Text**: `#adb5bd`

### 背景颜色
- **Page Background**: `linear-gradient(135deg, #ffffff 0%, #f8f9fa 50%, #ffffff 100%)`
- **Card Background**: `linear-gradient(145deg, #ffffff 0%, #f8f9fa 100%)`
- **Sider Background**: `linear-gradient(180deg, #1f1f1f 0%, #2d2d2d 100%)`

## 阴影系统

```css
--shadow-light: 0 4px 20px rgba(0, 0, 0, 0.08);
--shadow-medium: 0 8px 30px rgba(0, 0, 0, 0.12);
--shadow-heavy: 0 15px 40px rgba(0, 0, 0, 0.15);
```

每个卡片还包含内部高光:
```css
inset 0 1px 0 rgba(255, 255, 255, 0.9)
```

## 圆角系统

- **大圆角**: 12px (用于卡片、容器)
- **小圆角**: 8px (用于按钮、输入框、菜单项)

## 动画效果

### fadeInUp
```css
@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(30px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
```

### Hover 效果
- 卡片: `translateY(-3px)` + 增强阴影
- 按钮: `translateY(-2px)` + 增强阴影
- 菜单项: `translateX(3px)` + 背景高亮

## 测试清单

### 功能测试
- [ ] 登录页面正常显示
- [ ] 主布局正常渲染
- [ ] 侧边栏菜单可点击
- [ ] Dashboard 页面加载正常
- [ ] 统计卡片显示正确
- [ ] 图表正常渲染
- [ ] 表格数据显示正常

### 视觉测试
- [ ] 登录页背景为白色渐变
- [ ] 登录卡片有玻璃态效果
- [ ] 主按钮为黑色渐变
- [ ] 侧边栏为深色
- [ ] 卡片有 hover 动画
- [ ] 统计卡片顶部有彩色横条
- [ ] 字体为 Inter
- [ ] 圆角统一为 8px/12px

### 响应式测试
- [ ] 移动端布局正常
- [ ] 平板端布局正常
- [ ] 桌面端布局正常

## 构建和运行

```bash
# 开发模式
cd v2/hub/web
npm install
npm run dev

# 生产构建
npm run build
```

## 兼容性说明

- 所有样式与 Ant Design 5.x 兼容
- 使用 CSS 变量，需要现代浏览器支持
- 字体回退链确保在不支持 Inter 字体时正常显示
- 渐变效果在旧浏览器会降级为纯色

## 未来改进建议

1. 考虑添加暗色主题支持
2. 优化动画性能
3. 添加更多微交互效果
4. 统一其他页面 (Agents, Tasks, Remotes, Executions, Settings) 的样式
5. 考虑使用 CSS-in-JS 方案以获得更好的类型安全

## 注意事项

1. 确保 Inter 字体从 Google Fonts 正确加载
2. 玻璃态效果在某些浏览器可能需要 prefixes
3. 渐变背景可能在低端设备影响性能
4. 保持与 templates/ 风格的一致性
