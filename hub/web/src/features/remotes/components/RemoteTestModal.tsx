import React from 'react';
import { Alert, Button, Form, Input, Space, Typography } from 'antd';
import { ExperimentOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { Modal } from '../../../components/ui';
import type { RcloneRemote, RemoteTestResponse } from '../../../types';

export interface RemoteTestModalProps {
  isOpen: boolean;
  remote: RcloneRemote | null;
  testPath: string;
  submitting: boolean;
  result: RemoteTestResponse | null;
  onPathChange: (path: string) => void;
  onClose: () => void;
  onTest: () => void;
}

export const RemoteTestModal: React.FC<RemoteTestModalProps> = ({
  isOpen,
  remote,
  testPath,
  submitting,
  result,
  onPathChange,
  onClose,
  onTest,
}) => {
  const { t } = useTranslation();

  if (!remote) return null;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`${t('remotes.actions.test')}: ${remote.name}`}
      footer={
        <Space>
          <Button onClick={onClose} disabled={submitting}>
            {t('common.cancel')}
          </Button>
          <Button
            type="primary"
            icon={<ExperimentOutlined />}
            onClick={onTest}
            loading={submitting}
          >
            {submitting ? t('remotes.test.running') : t('remotes.actions.test')}
          </Button>
        </Space>
      }
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        {remote.type === 's3' && (
          <Form layout="vertical">
            <Form.Item
              label={t('remotes.test.pathLabel')}
              extra={t('remotes.test.pathPromptS3')}
            >
              <Input
                value={testPath}
                onChange={(e) => onPathChange(e.target.value)}
                placeholder={t('remotes.test.pathPlaceholder')}
                disabled={submitting}
              />
            </Form.Item>
          </Form>
        )}

        {result && (
          <Alert
            type={result.success ? 'success' : 'error'}
            showIcon
            message={`${result.success ? t('common.success') : t('common.failed')}${
              typeof result.duration_ms === 'number' ? ` (${result.duration_ms}ms)` : ''
            }`}
            description={
              <Space direction="vertical" style={{ width: '100%' }}>
                {result.message && <Typography.Text>{result.message}</Typography.Text>}
                {result.error && (
                  <Typography.Text code copyable>
                    {result.error}
                  </Typography.Text>
                )}
                {result.output && (
                  <Input.TextArea rows={6} value={result.output} readOnly />
                )}
              </Space>
            }
          />
        )}
      </Space>
    </Modal>
  );
};
