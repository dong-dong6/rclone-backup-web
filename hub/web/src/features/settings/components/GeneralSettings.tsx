import React from 'react';
import { Checkbox, Form, Input, InputNumber, Select } from 'antd';
import { ThemeSelector } from './ThemeSelector';
import type { Settings } from '../hooks';

export interface GeneralSettingsProps {
  settings: Settings;
  onSettingChange: <K extends keyof Settings>(key: K, value: Settings[K]) => void;
}

export const GeneralSettings: React.FC<GeneralSettingsProps> = ({
  settings,
  onSettingChange,
}) => {
  return (
    <Form layout="vertical" className="settings-form">
      <Form.Item label="Hub Name">
        <Input
          value={settings.hub_name}
          onChange={(e) => onSettingChange('hub_name', e.target.value)}
        />
      </Form.Item>

      <Form.Item label="Session Timeout (hours)">
        <InputNumber
          value={settings.session_timeout}
          min={1}
          style={{ width: '100%' }}
          onChange={(value) => onSettingChange('session_timeout', value || 0)}
        />
      </Form.Item>

      <Form.Item label="Log Level">
        <Select
          value={settings.log_level}
          onChange={(value) => onSettingChange('log_level', value)}
          options={[
            { value: 'debug', label: 'Debug' },
            { value: 'info', label: 'Info' },
            { value: 'warn', label: 'Warning' },
            { value: 'error', label: 'Error' },
          ]}
        />
      </Form.Item>

      <ThemeSelector />

      <Form.Item>
        <Checkbox
          checked={settings.enable_metrics}
          onChange={(e) => onSettingChange('enable_metrics', e.target.checked)}
        >
          Enable Metrics
        </Checkbox>
      </Form.Item>
    </Form>
  );
};
