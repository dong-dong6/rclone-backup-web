# TypeScript 配置文件详解

## 📝 为什么需要两个 tsconfig 文件？

在 Vite + TypeScript 项目中，我们需要两个不同的 TypeScript 配置文件来处理不同的代码上下文：

### 1. tsconfig.json - 应用代码配置

**作用**：配置浏览器端运行的应用代码

```json
{
  "compilerOptions": {
    "target": "ES2020",           // 编译目标：现代浏览器
    "lib": ["ES2020", "DOM"],     // 可用的API：浏览器DOM
    "jsx": "react-jsx",           // JSX处理：React
    "module": "ESNext",           // 模块系统：ES Modules
    "moduleResolution": "bundler" // 模块解析：打包工具
  },
  "include": ["src"],             // 包含：源代码目录
  "references": [                 // 引用：其他配置
    { "path": "./tsconfig.node.json" }
  ]
}
```

**这个配置处理**：
- React组件（.tsx文件）
- 业务逻辑（.ts文件）
- 工具函数
- API调用
- 状态管理

### 2. tsconfig.node.json - Node.js工具配置

**作用**：配置Node.js环境运行的构建工具代码

```json
{
  "compilerOptions": {
    "composite": true,     // 启用项目引用
    "skipLibCheck": true,  // 跳过库类型检查（加速）
    "module": "ESNext",    // 模块系统：ES Modules
    "moduleResolution": "bundler", // 模块解析：打包工具
    "allowSyntheticDefaultImports": true, // 允许默认导入
    "strict": true         // 严格模式
  },
  "include": ["vite.config.ts"] // 只包含Vite配置
}
```

**这个配置处理**：
- vite.config.ts（构建配置）
- 其他Node.js脚本
- 构建时插件

## 🤔 为什么要分离？

### 环境差异

| 特性 | 应用代码 (tsconfig.json) | 构建工具 (tsconfig.node.json) |
|-----|-------------------------|------------------------------|
| 运行环境 | 浏览器 | Node.js |
| 可用API | DOM, Window, fetch | fs, path, process |
| 模块系统 | ES Modules | ES Modules / CommonJS |
| 编译目标 | ES2020 (浏览器) | 当前Node版本 |

### 实际例子

```typescript
// ❌ 在 src/App.tsx 中（浏览器环境）
import fs from 'fs'; // 错误！浏览器没有fs

// ✅ 在 vite.config.ts 中（Node环境）
import fs from 'fs'; // 正确！Node.js有fs
```

## 🐛 常见错误

### 错误：ENOENT: no such file or directory, open 'tsconfig.node.json'

**原因**：文件缺失

**解决**：
1. 创建 tsconfig.node.json
2. 添加标准配置
3. 确保主配置引用它

### 错误：Cannot find module 'vite.config.ts'

**原因**：tsconfig.node.json 的 include 配置错误

**解决**：
```json
{
  "include": ["vite.config.ts"] // 确保包含正确文件
}
```

### 错误：Module '"fs"' has no exported member 'readFileSync'

**原因**：在浏览器代码中使用Node.js API

**解决**：
- 将Node.js代码移到vite.config.ts
- 或使用Vite的环境变量系统

## 📦 Docker构建中的作用

在Docker构建过程中：

```dockerfile
# 第一阶段：构建
COPY . .              # 复制所有文件（包括两个tsconfig）
RUN npm run build     # Vite读取两个配置文件
                      # → 编译vite.config.ts
                      # → 编译src/**/*.tsx
                      # → 生成dist目录
```

**构建流程**：
1. Vite启动
2. 读取vite.config.ts（使用tsconfig.node.json）
3. 编译应用代码（使用tsconfig.json）
4. 打包输出到dist

## 🔧 最佳实践

### 1. 始终提交两个配置文件

```bash
git add tsconfig.json tsconfig.node.json
git commit -m "chore: add TypeScript configs"
```

### 2. 保持配置同步

如果修改了模块解析策略，两个文件都要更新：

```json
// 两个文件都要改
"moduleResolution": "bundler"
```

### 3. 使用项目模板

创建新项目时使用官方模板：

```bash
npm create vite@latest my-app -- --template react-ts
```

这会自动生成正确的配置文件。

## 📊 配置对比

| 文件 | 用途 | 包含文件 | 目标环境 |
|-----|-----|---------|---------|
| tsconfig.json | 应用代码 | src/**/* | 浏览器 |
| tsconfig.node.json | 构建配置 | vite.config.ts | Node.js |

## 💡 调试技巧

### 验证配置

```bash
# 检查应用代码
npx tsc --noEmit

# 检查构建配置
npx tsc --project tsconfig.node.json --noEmit
```

### 查看实际配置

```bash
# 显示最终配置
npx tsc --showConfig
```

## 🎯 总结

- **两个配置文件是必要的**：分别处理浏览器和Node.js代码
- **都必须存在**：缺少任何一个都会导致构建失败
- **都要提交到Git**：确保CI/CD和Docker构建能够成功
- **保持简单**：使用标准配置，除非有特殊需求