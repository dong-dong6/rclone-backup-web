import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate, Link } from 'react-router-dom';
import {
  LayoutDashboard,
  Cloud,
  ListChecks,
  Database,
  History,
  Settings,
  User,
  LogOut,
  Bell,
  Globe,
  Menu as MenuIcon,
  ChevronLeft,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import Dashboard from './pages/Dashboard';
import Agents from './pages/Agents';
import Tasks from './pages/Tasks';
import Remotes from './pages/Remotes';
import Executions from './pages/Executions';
import SettingsPage from './pages/Settings';
import Login from './pages/Login';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { SSEProvider } from './contexts/SSEContext';
import ProtectedRoute from './components/ProtectedRoute';

import './App.css';

const AppLayout: React.FC = () => {
  const [collapsed, setCollapsed] = useState(window.innerWidth < 768);
  const { user, logout } = useAuth();
  const { t, i18n } = useTranslation();
  const [selectedKey, setSelectedKey] = useState('dashboard');

  useEffect(() => {
    const path = window.location.pathname.substring(1);
    setSelectedKey(path || 'dashboard');
  }, []);

  const menuItems = [
    { key: 'dashboard', icon: <LayoutDashboard size={20} />, label: t('menu.dashboard'), path: '/dashboard' },
    { key: 'agents', icon: <Cloud size={20} />, label: t('menu.agents'), path: '/agents' },
    { key: 'tasks', icon: <ListChecks size={20} />, label: t('menu.tasks'), path: '/tasks' },
    { key: 'remotes', icon: <Database size={20} />, label: t('menu.remotes'), path: '/remotes' },
    { key: 'executions', icon: <History size={20} />, label: t('menu.executions'), path: '/executions' },
    { key: 'settings', icon: <Settings size={20} />, label: t('menu.settings'), path: '/settings' },
  ];

  return (
    <div className={`app-layout ${collapsed ? 'collapsed' : ''}`}>
      <aside className="sidebar neu-card">
        <div className="sidebar-header">
          <div className="logo">
            {collapsed ? 'RB' : t('app.title')}
          </div>
          <button className="neu-button-icon toggle-button" onClick={() => setCollapsed(!collapsed)}>
            {collapsed ? <MenuIcon size={20} /> : <ChevronLeft size={20} />}
          </button>
        </div>
        <nav className="menu">
          {menuItems.map(item => (
            <Link
              key={item.key}
              to={item.path}
              className={`menu-item neu-button ${selectedKey === item.key ? 'active' : ''}`}
              onClick={() => setSelectedKey(item.key)}
            >
              {item.icon}
              {!collapsed && <span className="menu-label">{item.label}</span>}
            </Link>
          ))}
        </nav>
      </aside>
      <main className="main-content">
        <header className="header neu-card">
          <h1 className="page-title">
            {menuItems.find(item => item.key === selectedKey)?.label || t('app.title')}
          </h1>
          <div className="header-actions">
            <div className="neu-badge-group">
              <button className="neu-button-icon">
                <Bell size={20} />
                <span className="neu-badge-dot"></span>
              </button>
            </div>
            <div className="user-menu">
              <button className="neu-button user-button">
                <User size={20} />
                <span>{user?.name || 'Admin'}</span>
              </button>
            </div>
            <button className="neu-button-icon" onClick={logout}>
              <LogOut size={20} />
            </button>
          </div>
        </header>
        <div className="content-wrapper">
          <Routes>
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/agents" element={<Agents />} />
            <Route path="/tasks" element={<Tasks />} />
            <Route path="/remotes" element={<Remotes />} />
            <Route path="/executions" element={<Executions />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="/" element={<Navigate to="/dashboard" />} />
          </Routes>
        </div>
      </main>
    </div>
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