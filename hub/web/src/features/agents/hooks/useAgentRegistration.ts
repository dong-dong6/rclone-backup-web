import { useState, useCallback } from 'react';
import { agentsApi } from '../../../services';
import type { AgentRegistrationConfig } from '../../../types';

const defaultConfig: AgentRegistrationConfig = {
  agent_name: '',
  run_as_root: false,
  log_level: 'info',
  enable_api: false,
  api_port: 9092,
};

export function useAgentRegistration() {
  const [config, setConfig] = useState<AgentRegistrationConfig>(defaultConfig);
  const [token, setToken] = useState('');
  const [installCommand, setInstallCommand] = useState('');
  const [tokenExpiry, setTokenExpiry] = useState<Date | null>(null);
  const [showConfigForm, setShowConfigForm] = useState(true);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const updateConfig = useCallback(<K extends keyof AgentRegistrationConfig>(
    key: K,
    value: AgentRegistrationConfig[K]
  ) => {
    setConfig(prev => ({ ...prev, [key]: value }));
  }, []);

  const generateToken = useCallback(async (): Promise<boolean> => {
    setLoading(true);
    setError(null);
    try {
      const response = await agentsApi.generateRegistrationToken(config);
      setToken(response.token);
      setInstallCommand(response.install_command);

      // Set token expiry (24 hours from now)
      const expiry = new Date();
      expiry.setHours(expiry.getHours() + 24);
      setTokenExpiry(expiry);
      setShowConfigForm(false);

      return true;
    } catch (err) {
      console.error('Failed to generate token:', err);
      setError(err instanceof Error ? err.message : 'Token generation failed');
      return false;
    } finally {
      setLoading(false);
    }
  }, [config]);

  const reset = useCallback(() => {
    setConfig(defaultConfig);
    setToken('');
    setInstallCommand('');
    setTokenExpiry(null);
    setShowConfigForm(true);
    setError(null);
  }, []);

  const goBackToForm = useCallback(() => {
    setShowConfigForm(true);
  }, []);

  return {
    config,
    token,
    installCommand,
    tokenExpiry,
    showConfigForm,
    loading,
    error,
    updateConfig,
    generateToken,
    reset,
    goBackToForm,
  };
}
