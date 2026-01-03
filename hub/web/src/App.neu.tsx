import React, { useState, useEffect } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Dashboard from './pages/DashboardComplete';
import Agents from './pages/AgentsEnhanced';
import Tasks from './pages/Tasks';
import Remotes from './pages/Remotes';
import Executions from './pages/Executions';
import Settings from './pages/Settings';
import Login from './pages/LoginBW';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { SSEProvider } from './contexts/SSEContext';
import { ThemeProvider, useTheme } from './contexts/ThemeContext';
import ProtectedRoute from './components/ProtectedRoute';
import './styles/neumorphism-bw.css';
import './App.bw.css';

const AppLayout: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
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

  const menuItems = [
    {
      key: 'dashboard',
      icon: '📊',
      label: t('menu.dashboard'),
      path: '/dashboard',
    },
    {
      key: 'agents',
      icon: '🖥️',
      label: t('menu.agents'),
      path: '/agents',
    },
    {
      key: 'tasks',
      icon: '📅',
      label: t('menu.tasks'),
      path: '/tasks',
    },
    {
      key: 'remotes',
      icon: '☁️',
      label: t('menu.remotes'),
      path: '/remotes',
    },
    {
      key: 'executions',
      icon: '📝',
      label: t('menu.executions'),
      path: '/executions',
    },
    {
      key: 'settings',
      icon: '⚙️',
      label: t('menu.settings'),
      path: '/settings',
    },
  ];

  const toggleLanguage = () => {
    const newLang = i18n.language === 'zh' ? 'en' : 'zh';
    i18n.changeLanguage(newLang);
    localStorage.setItem('language', newLang);
  };

  return (
    <div className="app-container">
      {/* Mobile Menu Overlay */}
      {mobileMenuOpen && (
        <div 
          className="mobile-overlay"
          onClick={() => setMobileMenuOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside className={`sidebar ${collapsed ? 'collapsed' : ''} ${mobileMenuOpen ? 'mobile-open' : ''}`}>
        <div className="sidebar-header">
          <div className="logo">
            {collapsed ? '🔄' : '🔄 Rclone Backup'}
          </div>
          <button 
            className="neu-button collapse-btn desktop-only"
            onClick={() => setCollapsed(!collapsed)}
          >
            {collapsed ? '→' : '←'}
          </button>
        </div>

        <nav className="sidebar-menu">
          {menuItems.map(item => (
            <a
              key={item.key}
              href={item.path}
              className={`menu-item neu-card-flat ${selectedKey === item.key ? 'active' : ''}`}
              onClick={(e) => {
                e.preventDefault();
                setSelectedKey(item.key);
                window.location.href = item.path;
                setMobileMenuOpen(false);
              }}
            >
              <span className="menu-icon">{item.icon}</span>
              {!collapsed && <span className="menu-label">{item.label}</span>}
            </a>
          ))}
        </nav>

        <div className="sidebar-footer">
          {!collapsed && (
            <div className="user-info neu-card-inset">
              <div className="user-avatar">👤</div>
              <div className="user-details">
                <div className="user-name">{user?.name || 'Admin'}</div>
                <div className="user-role">{t('user.role.admin')}</div>
              </div>
            </div>
          )}
          <button 
            className="neu-button logout-btn"
            onClick={logout}
            title={t('user.logout')}
          >
            {collapsed ? '🚪' : `🚪 ${t('user.logout')}`}
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <div className={`main-content ${collapsed ? 'expanded' : ''}`}>
        {/* Header */}
        <header className="app-header neu-card-flat">
          <button
            className="neu-button mobile-menu-btn mobile-only"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          >
            ☰
          </button>

          <h1 className="app-title">{t('app.title')}</h1>

          <div className="header-actions">
            {/* Language Switcher */}
            <button
              className="neu-button lang-switch"
              onClick={toggleLanguage}
              title={t('settings.language')}
            >
              {i18n.language === 'zh' ? '🇨🇳 中文' : '🇬🇧 English'}
            </button>

            {/* Theme Switcher */}
            <button
              className="neu-button theme-switch"
              onClick={() => {
                const currentTheme = document.documentElement.getAttribute('data-theme');
                const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
                document.documentElement.setAttribute('data-theme', newTheme);
                localStorage.setItem('theme', newTheme);
              }}
              title={t('settings.theme')}
            >
              🌓
            </button>

            {/* Notifications */}
            <div className="notification-wrapper">
              <button className="neu-button notification-btn">
                🔔
                <span className="notification-badge">5</span>
              </button>
            </div>
          </div>
        </header>

        {/* Page Content */}
        <main className="page-content">
          <Routes>
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/agents" element={<Agents />} />
            <Route path="/tasks" element={<Tasks />} />
            <Route path="/remotes" element={<Remotes />} />
            <Route path="/executions" element={<Executions />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/" element={<Navigate to="/dashboard" />} />
          </Routes>
        </main>
      </div>
    </div>
  );
};

const App: React.FC = () => {
  useEffect(() => {
    // Load saved theme
    const savedTheme = localStorage.getItem('theme') || 'light';
    document.documentElement.setAttribute('data-theme', savedTheme);

    // Load saved language
    const savedLang = localStorage.getItem('language') || 'zh';
    // This will be handled by i18n provider
  }, []);

  return (
    <ThemeProvider>
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
    </ThemeProvider>
  );
};

export default App;