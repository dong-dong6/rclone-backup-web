import React from 'react';
import { Button, Card, Empty } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

export interface AgentEmptyStateProps {
  onRegister: () => void;
}

export const AgentEmptyState: React.FC<AgentEmptyStateProps> = ({ onRegister }) => {
  const { t } = useTranslation();

  return (
    <Card>
      <Empty
        description={
          <span>
            <strong>{t('agents.no_agents')}</strong>
            <small style={{ display: 'block', marginTop: 4 }}>
              {t('agents.no_agents_description')}
            </small>
          </span>
        }
      >
        <Button type="primary" icon={<PlusOutlined />} onClick={onRegister}>
          {t('agents.register.first_agent')}
        </Button>
      </Empty>
    </Card>
  );
};
