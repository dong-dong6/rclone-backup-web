import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconSettings, IconDatabase, IconShield, IconBell } from '@tabler/icons-react';

export type SettingsTab = 'general' | 'security' | 'database' | 'notifications';

interface TabItem {
  key: SettingsTab;
  labelKey: string;
  icon: React.FC<{ size?: number; className?: string }>;
}

const tabs: TabItem[] = [
  { key: 'general', labelKey: 'settings.tabs.general', icon: IconSettings },
  { key: 'security', labelKey: 'settings.tabs.security', icon: IconShield },
  { key: 'database', labelKey: 'settings.tabs.database', icon: IconDatabase },
  { key: 'notifications', labelKey: 'settings.tabs.notifications', icon: IconBell },
];

export interface SettingsTabsProps {
  activeTab: SettingsTab;
  onTabChange: (tab: SettingsTab) => void;
}

export const SettingsTabs: React.FC<SettingsTabsProps> = ({
  activeTab,
  onTabChange,
}) => {
  const { t } = useTranslation();

  return (
    <div className="nav nav-pills flex-column">
      {tabs.map(({ key, labelKey, icon: Icon }) => (
        <button
          key={key}
          className={`nav-link ${activeTab === key ? 'active' : ''}`}
          onClick={() => onTabChange(key)}
        >
          <Icon size={16} className="me-2" />
          {t(labelKey)}
        </button>
      ))}
    </div>
  );
};
