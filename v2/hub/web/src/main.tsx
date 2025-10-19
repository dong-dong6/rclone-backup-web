import React from 'react';
import ReactDOM from 'react-dom/client';
import { ConfigProvider, theme } from 'antd';
import App from './App.neu'; // 使用新拟态版本
import './i18n'; // 导入 i18n 配置
import './styles/refined.css'; // 第一阶段：精致化样式
import './styles/true-neumorphism.css'; // 第二阶段：真正的新拟态
import './index.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: '#1890ff',
        },
      }}
    >
      <App />
    </ConfigProvider>
  </React.StrictMode>
);