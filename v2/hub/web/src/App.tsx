import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { Layout, Menu, Avatar, Dropdown, Space, Badge } from 'antd';
import {
  DashboardOutlined,
  CloudServerOutlined,
  ScheduleOutlined,
  DatabaseOutlined,
  HistoryOutlined,
  SettingOutlined,
  UserOutlined,
  LogoutOutlined,
  BellOutlined,
  GlobalOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import Dashboard from './pages/Dashboard';
import Agents from './pages/Agents';
import Tasks from './pages/Tasks';
import Remotes from './pages/Remotes';
import Executions from './pages/Executions';
import Settings from './pages/Settings';
import Login from './pages/Login';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { SSEProvider } from './contexts/SSEContext';
import ProtectedRoute from './components/ProtectedRoute';
import './App.css';

const { Header, Sider, Content } = Layout;

const AppLayout: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const { user, logout } = useAuth();
  const { t, i18n } = useTranslation();
  const [selectedKey, setSelectedKey] = useState('dashboard');

  useEffect(() => {
    const path = window.location.pathname;
    if (path.includes('agents')) setSelectedKey('agents');
    else if (path.includes('tasks')) setSelectedKey('tasks');
    else if (path.includes('remotes')) setSelectedKey('remotes');
    else if (path.includes('executions')) setSelectedKey('executions');
    else if (path.includes('settings')) setSelectedKey('settings');
    else setSelectedKey('dashboard');
  }, []);

  const userMenu = (
    <Menu>
      <Menu.Item key="profile" icon={<UserOutlined />}>
        {t('menu.profile')}
      </Menu.Item>
      <Menu.Item key="language" icon={<GlobalOutlined />}>
        <Menu>
          <Menu.Item key="zh" onClick={() => i18n.changeLanguage('zh')}>
            中文
          </Menu.Item>
          <Menu.Item key="en" onClick={() => i18n.changeLanguage('en')}>
            English
          </Menu.Item>
        </Menu>
      </Menu.Item>
      <Menu.Item key="logout" icon={<LogoutOutlined />} onClick={logout}>
        {t('menu.logout')}
      </Menu.Item>
    </Menu>
  );

  const menuItems = [
    {
      key: 'dashboard',
      icon: <DashboardOutlined />,
      label: t('menu.dashboard'),
      path: '/dashboard',
    },
    {
      key: 'agents',
      icon: <CloudServerOutlined />,
      label: t('menu.agents'),
      path: '/agents',
    },
    {
      key: 'tasks',
      icon: <ScheduleOutlined />,
      label: t('menu.tasks'),
      path: '/tasks',
    },
    {
      key: 'remotes',
      icon: <DatabaseOutlined />,
      label: t('menu.remotes'),
      path: '/remotes',
    },
    {
      key: 'executions',
      icon: <HistoryOutlined />,
      label: t('menu.executions'),
      path: '/executions',
    },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: t('menu.settings'),
      path: '/settings',
    },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider
        collapsible
        collapsed={collapsed}
        onCollapse={setCollapsed}
        style={{
          overflow: 'auto',
          height: '100vh',
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
        }}
      >
        <div className="logo">
          {collapsed ? 'RB' : t('app.title')}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems.map(item => ({
            key: item.key,
            icon: item.icon,
            label: item.label,
            onClick: () => {
              setSelectedKey(item.key);
              window.location.href = item.path;
            },
          }))}
        />
      </Sider>
      <Layout style={{ marginLeft: collapsed ? 80 : 200, transition: 'all 0.2s' }}>
        <Header
          style={{
            padding: '0 24px',
            background: '#fff',
            boxShadow: '0 1px 4px rgba(0,0,0,0.08)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <h1 style={{ margin: 0, fontSize: '20px' }}>
            {t('app.title')}
          </h1>
          <Space size="large">
            <Badge count={5} size="small">
              <BellOutlined style={{ fontSize: '18px', cursor: 'pointer' }} />
            </Badge>
            <Dropdown overlay={userMenu} placement="bottomRight">
              <Space style={{ cursor: 'pointer' }}>
                <Avatar icon={<UserOutlined />} />
                <span>{user?.name || 'Admin'}</span>
              </Space>
            </Dropdown>
          </Space>
        </Header>
        <Content style={{ margin: '24px', minHeight: 'calc(100vh - 112px)' }}>
          <Routes>
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/agents" element={<Agents />} />
            <Route path="/tasks" element={<Tasks />} />
            <Route path="/remotes" element={<Remotes />} />
            <Route path="/executions" element={<Executions />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/" element={<Navigate to="/dashboard" />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  );
};

const App: React.FC = () => {
  return (
    <AuthProvider>
      <Router>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/*"
            element={
              <ProtectedRoute>
                <SSEProvider>
                  <AppLayout />
                </SSEProvider>
              </ProtectedRoute>
            }
          />
        </Routes>
      </Router>
    </AuthProvider>
  );
};

export default App;