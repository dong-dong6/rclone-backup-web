import React from 'react';
import { Button, Card, Empty } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

export interface RemoteEmptyStateProps {
  onCreate: () => void;
}

export const RemoteEmptyState: React.FC<RemoteEmptyStateProps> = ({ onCreate }) => {
  const { t } = useTranslation();

  return (
    <Card>
      <Empty description={t('remotes.list.empty')}>
        <Button type="primary" icon={<PlusOutlined />} onClick={onCreate}>
          {t('remotes.create.title')}
        </Button>
      </Empty>
    </Card>
  );
};
