import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate, Link, useLocation } from 'react-router-dom';
import {
  IconDashboard,
  IconCloud,
  IconListCheck,
  IconDatabase,
  IconHistory,
  IconSettings,
  IconUser,
  IconLogout,
  IconBell,
  IconMenu2,
  IconChevronLeft,
} from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import Dashboard from './pages/Dashboard';
import Agents from './pages/Agents';
import Tasks from './pages/Tasks';
import Remotes from './pages/Remotes';
import Executions, { ExecutionDetail } from './pages/Executions';
import SettingsPage from './pages/Settings';
import Login from './pages/Login';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { SSEProvider } from './contexts/SSEContext';
import ProtectedRoute from './components/ProtectedRoute';

import './App.css';

const AppLayout: React.FC = () => {
  const { user, logout } = useAuth();
  const { t, i18n } = useTranslation();
  const [selectedKey, setSelectedKey] = useState('dashboard');
  const location = useLocation();

  useEffect(() => {
    const pathSegment = location.pathname.split('/')[1];
    setSelectedKey(pathSegment || 'dashboard');
  }, [location.pathname]);

  const menuItems = [
    { key: 'dashboard', icon: <IconDashboard size={20} />, label: t('menu.dashboard'), path: '/dashboard' },
    { key: 'agents', icon: <IconCloud size={20} />, label: t('menu.agents'), path: '/agents' },
    { key: 'tasks', icon: <IconListCheck size={20} />, label: t('menu.tasks'), path: '/tasks' },
    { key: 'remotes', icon: <IconDatabase size={20} />, label: t('menu.remotes'), path: '/remotes' },
    { key: 'executions', icon: <IconHistory size={20} />, label: t('menu.executions'), path: '/executions' },
    { key: 'settings', icon: <IconSettings size={20} />, label: t('menu.settings'), path: '/settings' },
  ];

  return (
    <div className="page">
      <div className="page-wrapper">
        {/* Navbar */}
        <header className="navbar navbar-expand-md navbar-light d-print-none">
          <div className="container-xl">
            <button className="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbar-menu">
              <span className="navbar-toggler-icon"></span>
            </button>
            <h1 className="navbar-brand navbar-brand-autodark d-none-navbar-horizontal pe-0 pe-md-3">
              <a href=".">
                {t('app.title')}
              </a>
            </h1>
            <div className="navbar-nav flex-row order-md-last">
              <div className="nav-item dropdown">
                <a href="#" className="nav-link d-flex lh-1 text-reset p-0" data-bs-toggle="dropdown">
                  <span className="avatar avatar-sm" style={{backgroundImage: 'url(./static/avatars/000m.jpg)'}}></span>
                  <div className="d-none d-xl-block ps-2">
                    <div>{user?.name || 'Admin'}</div>
                    <div className="mt-1 small text-muted">Administrator</div>
                  </div>
                </a>
                <div className="dropdown-menu dropdown-menu-end dropdown-menu-arrow">
                  <a href="#" className="dropdown-item">Profile</a>
                  <a href="#" className="dropdown-item">Settings</a>
                  <div className="dropdown-divider"></div>
                  <a href="#" className="dropdown-item" onClick={logout}>Logout</a>
                </div>
              </div>
            </div>
            <div className="collapse navbar-collapse" id="navbar-menu">
              <div className="d-flex flex-column flex-md-row flex-fill align-items-stretch align-items-md-center">
                <ul className="navbar-nav">
                  {menuItems.map(item => (
                    <li key={item.key} className="nav-item">
                      <Link
                        to={item.path}
                        className={`nav-link ${selectedKey === item.key ? 'active' : ''}`}
                        onClick={() => setSelectedKey(item.key)}
                      >
                        <span className="nav-link-icon d-md-none d-lg-inline-block">
                          {item.icon}
                        </span>
                        <span className="nav-link-title">
                          {item.label}
                        </span>
                      </Link>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        </header>

        {/* Page header */}
        <div className="page-header d-print-none">
          <div className="container-xl">
            <div className="row g-2 align-items-center">
              <div className="col">
                <h2 className="page-title">
                  {menuItems.find(item => item.key === selectedKey)?.label || t('app.title')}
                </h2>
              </div>
            </div>
          </div>
        </div>

        {/* Page body */}
        <div className="page-body">
          <div className="container-xl">
            <Routes>
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/agents" element={<Agents />} />
              <Route path="/tasks" element={<Tasks />} />
              <Route path="/remotes" element={<Remotes />} />
              <Route path="/executions" element={<Executions />} />
              <Route path="/executions/:id" element={<ExecutionDetail />} />
              <Route path="/settings" element={<SettingsPage />} />
              <Route path="/" element={<Navigate to="/dashboard" />} />
            </Routes>
          </div>
        </div>
      </div>
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
