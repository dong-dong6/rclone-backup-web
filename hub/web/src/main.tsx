import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './i18n'; // 导入 i18n 配置
import './index.css';
import './App.css';

// 导入Tabler JavaScript
import '@tabler/core/dist/js/tabler.min.js';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);