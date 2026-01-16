import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { IconSettings, IconDatabase, IconShield, IconBell, IconMoon, IconSun, IconDeviceDesktop } from '@tabler/icons-react';
import { useTheme } from '../contexts/ThemeContext';

const Settings: React.FC = () => {
  const { t } = useTranslation();
  const { theme, setTheme } = useTheme();
  const [activeTab, setActiveTab] = useState('general');
  const [loading, setLoading] = useState(false);
  const [settings, setSettings] = useState({
    hub_name: 'Rclone Backup Hub',
    session_timeout: 24,
    log_level: 'info',
    enable_metrics: true,
  });

  const handleSave = async () => {
    setLoading(true);
    try {
      // TODO: Implement settings save
      console.log('Saving settings:', settings);
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 1000));
      alert(t('app.success'));
    } catch (error) {
      alert(t('errors.server'));
    } finally {
      setLoading(false);
    }
  };

  const tabs = [
    { key: 'general', label: t('settings.tabs.general'), icon: IconSettings },
    { key: 'security', label: t('settings.tabs.security'), icon: IconShield },
    { key: 'database', label: t('settings.tabs.database'), icon: IconDatabase },
    { key: 'notifications', label: t('settings.tabs.notifications'), icon: IconBell },
  ];

  return (
    <div className="row row-deck row-cards">
      <div className="col-12">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">{t('settings.title')}</h3>
          </div>
          <div className="card-body">
            <div className="row">
              <div className="col-md-3">
                <div className="nav nav-pills flex-column">
                  {tabs.map(tab => (
                    <button
                      key={tab.key}
                      className={`nav-link ${activeTab === tab.key ? 'active' : ''}`}
                      onClick={() => setActiveTab(tab.key)}
                    >
                      <tab.icon size={16} className="me-2" />
                      {tab.label}
                    </button>
                  ))}
                </div>
              </div>
              <div className="col-md-9">
                {activeTab === 'general' && (
                  <div>
                    <h4>General Settings</h4>
                    <div className="mb-3">
                      <label className="form-label">Hub Name</label>
                      <input
                        type="text"
                        className="form-control"
                        value={settings.hub_name}
                        onChange={(e) => setSettings({ ...settings, hub_name: e.target.value })}
                      />
                    </div>
                    <div className="mb-3">
                      <label className="form-label">Session Timeout (hours)</label>
                      <input
                        type="number"
                        className="form-control"
                        value={settings.session_timeout}
                        onChange={(e) => setSettings({ ...settings, session_timeout: parseInt(e.target.value) })}
                      />
                    </div>
                    <div className="mb-3">
                      <label className="form-label">Log Level</label>
                      <select
                        className="form-select"
                        value={settings.log_level}
                        onChange={(e) => setSettings({ ...settings, log_level: e.target.value })}
                      >
                        <option value="debug">Debug</option>
                        <option value="info">Info</option>
                        <option value="warn">Warning</option>
                        <option value="error">Error</option>
                      </select>
                    </div>
                    <div className="mb-3">
                      <label className="form-label">{t('settings.theme') || 'Theme'}</label>
                      <div className="row g-2">
                        <div className="col-4">
                          <label className={`form-selectgroup-item`}>
                            <input
                              type="radio"
                              name="theme"
                              value="light"
                              className="form-selectgroup-input"
                              checked={theme === 'light'}
                              onChange={() => setTheme('light')}
                            />
                            <span className="form-selectgroup-label d-flex align-items-center p-3 justify-content-center">
                              <IconSun size={18} className="me-2" />
                              Light
                            </span>
                          </label>
                        </div>
                        <div className="col-4">
                          <label className="form-selectgroup-item">
                            <input
                              type="radio"
                              name="theme"
                              value="dark"
                              className="form-selectgroup-input"
                              checked={theme === 'dark'}
                              onChange={() => setTheme('dark')}
                            />
                            <span className="form-selectgroup-label d-flex align-items-center p-3 justify-content-center">
                              <IconMoon size={18} className="me-2" />
                              Dark
                            </span>
                          </label>
                        </div>
                        <div className="col-4">
                          <label className="form-selectgroup-item">
                            <input
                              type="radio"
                              name="theme"
                              value="auto"
                              className="form-selectgroup-input"
                              checked={theme === 'auto'}
                              onChange={() => setTheme('auto')}
                            />
                            <span className="form-selectgroup-label d-flex align-items-center p-3 justify-content-center">
                              <IconDeviceDesktop size={18} className="me-2" />
                              Auto
                            </span>
                          </label>
                        </div>
                      </div>
                    </div>

                    <div className="mb-3">
                      <div className="form-check">
                        <input
                          type="checkbox"
                          className="form-check-input"
                          checked={settings.enable_metrics}
                          onChange={(e) => setSettings({ ...settings, enable_metrics: e.target.checked })}
                        />
                        <label className="form-check-label">Enable Metrics</label>
                      </div>
                    </div>
                  </div>
                )}

                {activeTab === 'security' && (
                  <div>
                    <h4>Security Settings</h4>
                    <p className="text-muted">Security configuration options will be displayed here.</p>
                  </div>
                )}

                {activeTab === 'database' && (
                  <div>
                    <h4>Database Settings</h4>
                    <p className="text-muted">Database configuration options will be displayed here.</p>
                  </div>
                )}

                {activeTab === 'notifications' && (
                  <div>
                    <h4>Notification Settings</h4>
                    <p className="text-muted">Notification configuration options will be displayed here.</p>
                  </div>
                )}

                <div className="mt-4">
                  <button
                    className="btn btn-primary"
                    onClick={handleSave}
                    disabled={loading}
                  >
                    {loading ? t('common.saving') : t('common.save')}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Settings;
