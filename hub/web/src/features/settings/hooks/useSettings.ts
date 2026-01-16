import { useState, useCallback } from 'react';

export interface Settings {
  hub_name: string;
  session_timeout: number;
  log_level: string;
  enable_metrics: boolean;
}

const defaultSettings: Settings = {
  hub_name: 'Rclone Backup Hub',
  session_timeout: 24,
  log_level: 'info',
  enable_metrics: true,
};

export function useSettings() {
  const [settings, setSettings] = useState<Settings>(defaultSettings);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateSetting = useCallback(<K extends keyof Settings>(
    key: K,
    value: Settings[K]
  ) => {
    setSettings(prev => ({ ...prev, [key]: value }));
  }, []);

  const saveSettings = useCallback(async (): Promise<boolean> => {
    setLoading(true);
    setError(null);
    try {
      // TODO: Implement actual API call
      console.log('Saving settings:', settings);
      await new Promise(resolve => setTimeout(resolve, 1000));
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save settings');
      return false;
    } finally {
      setLoading(false);
    }
  }, [settings]);

  const resetSettings = useCallback(() => {
    setSettings(defaultSettings);
  }, []);

  return {
    settings,
    loading,
    error,
    updateSetting,
    saveSettings,
    resetSettings,
  };
}
