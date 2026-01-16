import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconSun, IconMoon, IconDeviceDesktop } from '@tabler/icons-react';
import { useTheme, type Theme } from '../../../contexts/ThemeContext';

interface ThemeOption {
  value: Theme;
  label: string;
  icon: React.FC<{ size?: number; className?: string }>;
}

export const ThemeSelector: React.FC = () => {
  const { t } = useTranslation();
  const { theme, setTheme } = useTheme();

  const options: ThemeOption[] = [
    { value: 'light', label: 'Light', icon: IconSun },
    { value: 'dark', label: 'Dark', icon: IconMoon },
    { value: 'auto', label: 'Auto', icon: IconDeviceDesktop },
  ];

  return (
    <div className="mb-3">
      <label className="form-label">{t('settings.theme') || 'Theme'}</label>
      <div className="row g-2">
        {options.map(({ value, label, icon: Icon }) => (
          <div key={value} className="col-4">
            <label className="form-selectgroup-item">
              <input
                type="radio"
                name="theme"
                value={value}
                className="form-selectgroup-input"
                checked={theme === value}
                onChange={() => setTheme(value)}
              />
              <span className="form-selectgroup-label d-flex align-items-center p-3 justify-content-center">
                <Icon size={18} className="me-2" />
                {label}
              </span>
            </label>
          </div>
        ))}
      </div>
    </div>
  );
};
