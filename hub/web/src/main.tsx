import React from 'react';
import ReactDOM from 'react-dom/client';
import { App as AntApp, ConfigProvider } from 'antd';
import 'antd/dist/reset.css';
import App from './App';
import { ThemeProvider } from './contexts/ThemeContext';
import './i18n';
import './index.css';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ConfigProvider
      theme={{
        token: {
          colorPrimary: '#2563eb',
          borderRadius: 8,
          fontFamily: 'Inter, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
        },
      }}
    >
      <AntApp>
        <ThemeProvider>
          <App />
        </ThemeProvider>
      </AntApp>
    </ConfigProvider>
  </React.StrictMode>
);
