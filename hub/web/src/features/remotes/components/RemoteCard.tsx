import React from 'react';
import { Button, Card, Space, Tag, Tooltip, Typography } from 'antd';
import {
  DeleteOutlined,
  EditOutlined,
  LoadingOutlined,
  ExperimentOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { RcloneRemote } from '../../../types';

export interface RemoteCardProps {
  remote: RcloneRemote;
  testing?: boolean;
  onTest: (id: string) => void;
  onEdit: (remote: RcloneRemote) => void;
  onDelete: (id: string) => void;
}

export const RemoteCard: React.FC<RemoteCardProps> = ({
  remote,
  testing,
  onTest,
  onEdit,
  onDelete,
}) => {
  const { t } = useTranslation();

  const getTestStatusTag = () => {
    if (remote.last_test_success === true) {
      return <Tag color="green">{t('common.success')}</Tag>;
    }
    if (remote.last_test_success === false) {
      return <Tag color="red">{t('common.failed')}</Tag>;
    }
    return <Tag>{t('common.never')}</Tag>;
  };

  return (
    <Card
      title={
        <Space direction="vertical" size={4}>
          <Typography.Text strong>{remote.name}</Typography.Text>
          <Space size={4} wrap>
            {remote.type && <Tag>{remote.type}</Tag>}
            <Tooltip title={remote.last_test_error || remote.last_test_message || ''}>
              {getTestStatusTag()}
            </Tooltip>
          </Space>
        </Space>
      }
      extra={
        <Space>
          <Tooltip title={t('remotes.actions.test')}>
            <Button
              icon={testing ? <LoadingOutlined spin /> : <ExperimentOutlined />}
              onClick={() => onTest(remote.id)}
              disabled={testing}
            />
          </Tooltip>
          <Tooltip title={t('common.edit')}>
            <Button icon={<EditOutlined />} onClick={() => onEdit(remote)} />
          </Tooltip>
          <Tooltip title={t('common.delete')}>
            <Button danger icon={<DeleteOutlined />} onClick={() => onDelete(remote.id)} />
          </Tooltip>
        </Space>
      }
    >
      <Space direction="vertical" size={4}>
        <Typography.Text type="secondary">
          {t('remotes.list.columns.createdAt')}: {new Date(remote.created_at).toLocaleString()}
        </Typography.Text>
        <Typography.Text type="secondary">
          {t('remotes.list.columns.lastTest')}:{' '}
          {remote.last_test_at ? new Date(remote.last_test_at).toLocaleString() : t('common.never')}
        </Typography.Text>
      </Space>
    </Card>
  );
};
