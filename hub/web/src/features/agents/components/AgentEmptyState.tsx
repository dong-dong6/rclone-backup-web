import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconServer, IconPlus } from '@tabler/icons-react';

export interface AgentEmptyStateProps {
  onRegister: () => void;
}

export const AgentEmptyState: React.FC<AgentEmptyStateProps> = ({ onRegister }) => {
  const { t } = useTranslation();

  return (
    <div className="col-12">
      <div className="card">
        <div className="card-body text-center py-5">
          <IconServer className="mx-auto text-muted mb-4" size={64} />
          <h3 className="card-title">{t('agents.no_agents')}</h3>
          <p className="text-muted mb-4">{t('agents.no_agents_description')}</p>
          <button onClick={onRegister} className="btn btn-primary">
            <IconPlus size={16} className="me-1" />
            {t('agents.register.first_agent')}
          </button>
        </div>
      </div>
    </div>
  );
};
