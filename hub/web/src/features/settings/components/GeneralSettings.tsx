import React from 'react';
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
    <div>
      <h4>General Settings</h4>

      <div className="mb-3">
        <label className="form-label">Hub Name</label>
        <input
          type="text"
          className="form-control"
          value={settings.hub_name}
          onChange={(e) => onSettingChange('hub_name', e.target.value)}
        />
      </div>

      <div className="mb-3">
        <label className="form-label">Session Timeout (hours)</label>
        <input
          type="number"
          className="form-control"
          value={settings.session_timeout}
          onChange={(e) => onSettingChange('session_timeout', parseInt(e.target.value) || 0)}
        />
      </div>

      <div className="mb-3">
        <label className="form-label">Log Level</label>
        <select
          className="form-select"
          value={settings.log_level}
          onChange={(e) => onSettingChange('log_level', e.target.value)}
        >
          <option value="debug">Debug</option>
          <option value="info">Info</option>
          <option value="warn">Warning</option>
          <option value="error">Error</option>
        </select>
      </div>

      <ThemeSelector />

      <div className="mb-3">
        <div className="form-check">
          <input
            type="checkbox"
            className="form-check-input"
            id="enableMetrics"
            checked={settings.enable_metrics}
            onChange={(e) => onSettingChange('enable_metrics', e.target.checked)}
          />
          <label className="form-check-label" htmlFor="enableMetrics">
            Enable Metrics
          </label>
        </div>
      </div>
    </div>
  );
};
