import React, { useState } from 'react';
import { App, Button, Card, Space, Tabs } from 'antd';
import type { TabsProps } from 'antd';
import {
  BellOutlined,
  DatabaseOutlined,
  SaveOutlined,
  SecurityScanOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useSettings } from './hooks';
import {
  GeneralSettings,
  SecuritySettings,
  DatabaseSettings,
  NotificationSettings,
  type SettingsTab,
} from './components';

export const SettingsPage: React.FC = () => {
  const { t } = useTranslation();
  const { message } = App.useApp();
  const [activeTab, setActiveTab] = useState<SettingsTab>('general');
  const { settings, loading, updateSetting, saveSettings } = useSettings();

  const handleSave = async () => {
    const success = await saveSettings();
    if (success) {
      message.success(t('app.success'));
    } else {
      message.error(t('errors.server'));
    }
  };

  const tabItems: TabsProps['items'] = [
    {
      key: 'general',
      label: (
        <Space>
          <SettingOutlined />
          {t('settings.tabs.general')}
        </Space>
      ),
      children: (
        <GeneralSettings
          settings={settings}
          onSettingChange={updateSetting}
        />
      ),
    },
    {
      key: 'security',
      label: (
        <Space>
          <SecurityScanOutlined />
          {t('settings.tabs.security')}
        </Space>
      ),
      children: <SecuritySettings />,
    },
    {
      key: 'database',
      label: (
        <Space>
          <DatabaseOutlined />
          {t('settings.tabs.database')}
        </Space>
      ),
      children: <DatabaseSettings />,
    },
    {
      key: 'notifications',
      label: (
        <Space>
          <BellOutlined />
          {t('settings.tabs.notifications')}
        </Space>
      ),
      children: <NotificationSettings />,
    },
  ];

  return (
    <div className="rbw-page">
      <Card
        title={t('settings.title')}
        extra={
          <Button
            type="primary"
            icon={<SaveOutlined />}
            onClick={handleSave}
            loading={loading}
          >
            {loading ? t('common.saving') : t('common.save')}
          </Button>
        }
      >
        <Tabs
          activeKey={activeTab}
          items={tabItems}
          onChange={(key) => setActiveTab(key as SettingsTab)}
        />
      </Card>
    </div>
  );
};

export default SettingsPage;
