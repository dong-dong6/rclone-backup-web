import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { IconPlus, IconCopy, IconCheck, IconAlertCircle } from '@tabler/icons-react';
import { Modal } from '../../../components/ui';
import { useAgentRegistration } from '../hooks';
import { useClipboard } from '../../../hooks';

export interface AgentRegisterModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AgentRegisterModal: React.FC<AgentRegisterModalProps> = ({
  isOpen,
  onClose,
}) => {
  const { t } = useTranslation();
  const { copied, copy } = useClipboard();
  const {
    config,
    installCommand,
    tokenExpiry,
    showConfigForm,
    loading,
    updateConfig,
    generateToken,
    reset,
    goBackToForm,
  } = useAgentRegistration();

  const handleClose = () => {
    reset();
    onClose();
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      title={t('agents.register.title')}
      size="lg"
      footer={
        <>
          {!showConfigForm && (
            <button onClick={goBackToForm} className="btn btn-outline-secondary me-auto">
              {t('common.back')}
            </button>
          )}
          <button onClick={handleClose} className="btn btn-secondary">
            {t('common.close')}
          </button>
        </>
      }
    >
      {showConfigForm ? (
        /* Configuration Form */
        <>
          <div className="mb-3">
            <label className="form-label">{t('agents.register.agent_name')}</label>
            <input
              type="text"
              className="form-control"
              value={config.agent_name}
              onChange={(e) => updateConfig('agent_name', e.target.value)}
              placeholder={t('agents.register.agent_name_placeholder')}
            />
            <small className="text-muted">{t('agents.register.agent_name_hint')}</small>
          </div>

          <div className="mb-3">
            <label className="form-check">
              <input
                type="checkbox"
                className="form-check-input"
                checked={config.run_as_root}
                onChange={(e) => updateConfig('run_as_root', e.target.checked)}
              />
              <span className="form-check-label">{t('agents.register.run_as_root')}</span>
            </label>
            {config.run_as_root && (
              <div className="alert alert-warning mt-2 mb-0">
                <IconAlertCircle className="me-1" size={16} />
                {t('agents.register.root_warning')}
              </div>
            )}
          </div>

          <div className="mb-3">
            <label className="form-label">{t('agents.register.log_level')}</label>
            <select
              className="form-select"
              value={config.log_level}
              onChange={(e) => updateConfig('log_level', e.target.value)}
            >
              <option value="debug">DEBUG</option>
              <option value="info">INFO</option>
              <option value="warn">WARN</option>
              <option value="error">ERROR</option>
            </select>
          </div>

          <div className="mb-3">
            <label className="form-check">
              <input
                type="checkbox"
                className="form-check-input"
                checked={config.enable_api}
                onChange={(e) => updateConfig('enable_api', e.target.checked)}
              />
              <span className="form-check-label">{t('agents.register.enable_api')}</span>
            </label>
            {config.enable_api && (
              <div className="mt-2 ms-4">
                <label className="form-label mb-1">{t('agents.register.api_port')}</label>
                <input
                  type="number"
                  className="form-control"
                  value={config.api_port}
                  onChange={(e) => updateConfig('api_port', parseInt(e.target.value) || 9092)}
                  min={1024}
                  max={65535}
                />
              </div>
            )}
          </div>

          <div className="d-grid">
            <button onClick={generateToken} className="btn btn-primary" disabled={loading}>
              <IconPlus size={16} className="me-1" />
              {t('agents.register.generate_command')}
            </button>
          </div>
        </>
      ) : (
        /* Generated Install Command */
        <>
          {tokenExpiry && (
            <div className="alert alert-warning mb-3">
              <IconAlertCircle className="me-1" size={16} />
              {t('agents.register.token_expires', {
                time: tokenExpiry.toLocaleString(),
              })}
            </div>
          )}

          <div className="mb-3">
            <label className="form-label">{t('agents.register.install_command')}</label>
            <div className="position-relative">
              <pre
                className="bg-dark text-light p-3 rounded mb-0"
                style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}
              >
                <code>{installCommand}</code>
              </pre>
              <button
                onClick={() => copy(installCommand)}
                className="btn btn-sm btn-light position-absolute"
                style={{ top: '8px', right: '8px' }}
              >
                {copied ? (
                  <IconCheck size={16} className="text-success" />
                ) : (
                  <IconCopy size={16} />
                )}
                <span className="ms-1">{copied ? t('common.copied') : t('common.copy')}</span>
              </button>
            </div>
          </div>

          <div className="card bg-light">
            <div className="card-body">
              <h4 className="card-title">{t('agents.register.instructions')}</h4>
              <ol className="mb-0">
                <li>{t('agents.register.step1')}</li>
                <li>{t('agents.register.step2')}</li>
                <li>{t('agents.register.step3')}</li>
              </ol>
            </div>
          </div>
        </>
      )}
    </Modal>
  );
};
