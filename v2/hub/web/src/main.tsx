import React from 'react';
import ReactDOM from 'react-dom/client';
import { ConfigProvider, theme } from 'antd';
import App from './App'; // 使用标准版本
import './i18n'; // 导入 i18n 配置
import './index.css';
import './App.css';

// 配置Ant Design主题以匹配templates/base.html风格
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          // 主色调改为黑色渐变系
          colorPrimary: '#000000',
          colorSuccess: '#28a745',
          colorWarning: '#ffc107',
          colorError: '#dc3545',
          colorInfo: '#17a2b8',

          // 边框和圆角
          borderRadius: 8,
          borderRadiusLG: 12,

          // 字体
          fontFamily: "'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif",
          fontSize: 14,
          fontWeightStrong: 600,

          // 阴影
          boxShadow: '0 4px 20px rgba(0, 0, 0, 0.08)',
          boxShadowSecondary: '0 8px 30px rgba(0, 0, 0, 0.12)',

          // 颜色
          colorBgContainer: '#ffffff',
          colorBgElevated: '#ffffff',
          colorBorder: 'rgba(0, 0, 0, 0.1)',
          colorBorderSecondary: 'rgba(0, 0, 0, 0.06)',

          // 文本颜色
          colorText: '#212529',
          colorTextSecondary: '#6c757d',
          colorTextTertiary: '#adb5bd',
        },
        components: {
          Card: {
            borderRadiusLG: 12,
            boxShadowTertiary: '0 4px 20px rgba(0, 0, 0, 0.08), inset 0 1px 0 rgba(255, 255, 255, 0.9)',
          },
          Button: {
            borderRadius: 8,
            fontWeight: 600,
            controlHeight: 38,
            controlHeightLG: 45,
          },
          Input: {
            borderRadius: 8,
            controlHeight: 38,
          },
          Table: {
            borderRadius: 12,
            borderRadiusLG: 12,
          },
          Menu: {
            itemBorderRadius: 8,
          },
        },
      }}
    >
      <App />
    </ConfigProvider>
  </React.StrictMode>
);