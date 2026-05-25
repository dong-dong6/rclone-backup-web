import React, { useEffect, useMemo, useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate, Link, useLocation } from 'react-router-dom';
import {
  Avatar,
  Button,
  Dropdown,
  Layout,
  Menu,
  Space,
  Typography,
} from 'antd';
import type { MenuProps } from 'antd';
import {
  CloudOutlined,
  DashboardOutlined,
  DatabaseOutlined,
  HistoryOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  SettingOutlined,
  UnorderedListOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  DashboardPage,
  AgentsPage,
  TasksPage,
  RemotesPage,
  ExecutionsPage,
  ExecutionDetailPage,
  SettingsPage,
} from './features';
import Login from './pages/Login';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { SSEProvider } from './contexts/SSEContext';
import ProtectedRoute from './components/ProtectedRoute';

import './App.css';

const { Header, Sider, Content } = Layout;
const { Text, Title } = Typography;

const AppLayout: React.FC = () => {
  const { user, logout } = useAuth();
  const { t } = useTranslation();
  const [collapsed, setCollapsed] = useState(false);
  const [selectedKey, setSelectedKey] = useState('dashboard');
  const location = useLocation();

  useEffect(() => {
    const pathSegment = location.pathname.split('/')[1];
    setSelectedKey(pathSegment || 'dashboard');
  }, [location.pathname]);

  const menuItems = useMemo<MenuProps['items']>(
    () => [
      { key: 'dashboard', icon: <DashboardOutlined />, label: <Link to="/dashboard">{t('menu.dashboard')}</Link> },
      { key: 'agents', icon: <CloudOutlined />, label: <Link to="/agents">{t('menu.agents')}</Link> },
      { key: 'tasks', icon: <UnorderedListOutlined />, label: <Link to="/tasks">{t('menu.tasks')}</Link> },
      { key: 'remotes', icon: <DatabaseOutlined />, label: <Link to="/remotes">{t('menu.remotes')}</Link> },
      { key: 'executions', icon: <HistoryOutlined />, label: <Link to="/executions">{t('menu.executions')}</Link> },
      { key: 'settings', icon: <SettingOutlined />, label: <Link to="/settings">{t('menu.settings')}</Link> },
    ],
    [t]
  );

  const pageTitle = t(`menu.${selectedKey}`, { defaultValue: t('app.title') });

  const userMenu: MenuProps['items'] = [
    {
      key: 'profile',
      icon: <UserOutlined />,
      label: 'Profile',
    },
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: <Link to="/settings">Settings</Link>,
    },
    {
      type: 'divider',
    },
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: 'Logout',
      danger: true,
      onClick: logout,
    },
  ];

  return (
    <Layout className="app-shell">
      <Sider
        className="app-sider"
        collapsible
        collapsed={collapsed}
        trigger={null}
        breakpoint="lg"
        onBreakpoint={setCollapsed}
      >
        <div className="app-brand">
          <CloudOutlined className="app-brand-icon" />
          {!collapsed && <span>{t('app.title')}</span>}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={menuItems}
          onClick={({ key }) => setSelectedKey(key)}
        />
      </Sider>

      <Layout>
        <Header className="app-header">
          <Space size={16}>
            <Button
              type="text"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={() => setCollapsed((value) => !value)}
            />
            <Title level={4} className="app-page-title">
              {pageTitle}
            </Title>
          </Space>

          <Dropdown menu={{ items: userMenu }} placement="bottomRight">
            <Button type="text" className="app-user-button">
              <Space>
                <Avatar size="small" icon={<UserOutlined />} />
                <span className="app-user-meta">
                  <Text strong>{user?.name || 'Admin'}</Text>
                  <Text type="secondary">Administrator</Text>
                </span>
              </Space>
            </Button>
          </Dropdown>
        </Header>

        <Content className="app-content">
          <Routes>
            <Route path="/dashboard" element={<DashboardPage />} />
            <Route path="/agents" element={<AgentsPage />} />
            <Route path="/tasks" element={<TasksPage />} />
            <Route path="/remotes" element={<RemotesPage />} />
            <Route path="/executions" element={<ExecutionsPage />} />
            <Route path="/executions/:id" element={<ExecutionDetailPage />} />
            <Route path="/settings" element={<SettingsPage />} />
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
