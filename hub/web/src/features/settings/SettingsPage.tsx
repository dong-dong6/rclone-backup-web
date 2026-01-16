import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useSettings } from './hooks';
import {
  SettingsTabs,
  GeneralSettings,
  SecuritySettings,
  DatabaseSettings,
  NotificationSettings,
  type SettingsTab,
} from './components';

export const SettingsPage: React.FC = () => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');
  const { settings, loading, updateSetting, saveSettings } = useSettings();

  const handleSave = async () => {
    const success = await saveSettings();
    if (success) {
      alert(t('app.success'));
    } else {
      alert(t('errors.server'));
    }
  };

  const renderTabContent = () => {
    switch (activeTab) {
      case 'general':
        return (
          <GeneralSettings
            settings={settings}
            onSettingChange={updateSetting}
          />
        );
      case 'security':
        return <SecuritySettings />;
      case 'database':
        return <DatabaseSettings />;
      case 'notifications':
        return <NotificationSettings />;
      default:
        return null;
    }
  };

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
                <SettingsTabs
                  activeTab={activeTab}
                  onTabChange={setActiveTab}
                />
              </div>
              <div className="col-md-9">
                {renderTabContent()}

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

export default SettingsPage;
