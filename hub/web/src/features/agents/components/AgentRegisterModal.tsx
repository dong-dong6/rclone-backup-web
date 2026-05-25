import React from 'react';
import { Alert, Button, Card, Checkbox, Form, Input, InputNumber, Select, Space, Typography } from 'antd';
import { CheckOutlined, CopyOutlined, PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
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
        <Space>
          {!showConfigForm && (
            <Button onClick={goBackToForm}>
              {t('common.back')}
            </Button>
          )}
          <Button onClick={handleClose}>
            {t('common.close')}
          </Button>
        </Space>
      }
    >
      {showConfigForm ? (
        <Form layout="vertical">
          <Form.Item label={t('agents.register.agent_name')} extra={t('agents.register.agent_name_hint')}>
            <Input
              value={config.agent_name}
              onChange={(e) => updateConfig('agent_name', e.target.value)}
              placeholder={t('agents.register.agent_name_placeholder')}
            />
          </Form.Item>

          <Form.Item>
            <Checkbox
              checked={config.run_as_root}
              onChange={(e) => updateConfig('run_as_root', e.target.checked)}
            >
              {t('agents.register.run_as_root')}
            </Checkbox>
            {config.run_as_root && (
              <Alert
                type="warning"
                showIcon
                message={t('agents.register.root_warning')}
                style={{ marginTop: 8 }}
              />
            )}
          </Form.Item>

          <Form.Item label={t('agents.register.log_level')}>
            <Select
              value={config.log_level}
              onChange={(value) => updateConfig('log_level', value)}
              options={[
                { value: 'debug', label: 'DEBUG' },
                { value: 'info', label: 'INFO' },
                { value: 'warn', label: 'WARN' },
                { value: 'error', label: 'ERROR' },
              ]}
            />
          </Form.Item>

          <Form.Item>
            <Checkbox
              checked={config.enable_api}
              onChange={(e) => updateConfig('enable_api', e.target.checked)}
            >
              {t('agents.register.enable_api')}
            </Checkbox>
          </Form.Item>

          {config.enable_api && (
            <Form.Item label={t('agents.register.api_port')}>
              <InputNumber
                value={config.api_port}
                onChange={(value) => updateConfig('api_port', value || 9092)}
                min={1024}
                max={65535}
                style={{ width: '100%' }}
              />
            </Form.Item>
          )}

          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={generateToken}
            loading={loading}
            block
          >
            {t('agents.register.generate_command')}
          </Button>
        </Form>
      ) : (
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          {tokenExpiry && (
            <Alert
              type="warning"
              showIcon
              message={t('agents.register.token_expires', {
                time: tokenExpiry.toLocaleString(),
              })}
            />
          )}

          <div>
            <Typography.Text strong>{t('agents.register.install_command')}</Typography.Text>
            <Card
              size="small"
              style={{ marginTop: 8, background: '#111827' }}
              extra={
                <Button
                  icon={copied ? <CheckOutlined /> : <CopyOutlined />}
                  onClick={() => copy(installCommand)}
                >
                  {copied ? t('common.copied') : t('common.copy')}
                </Button>
              }
            >
              <Typography.Text
                code
                style={{ color: '#f9fafb', whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}
              >
                {installCommand}
              </Typography.Text>
            </Card>
          </div>

          <Card size="small" title={t('agents.register.instructions')}>
            <ol style={{ margin: 0, paddingLeft: 20 }}>
              <li>{t('agents.register.step1')}</li>
              <li>{t('agents.register.step2')}</li>
              <li>{t('agents.register.step3')}</li>
            </ol>
          </Card>
        </Space>
      )}
    </Modal>
  );
};
