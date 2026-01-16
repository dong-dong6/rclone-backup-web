import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Loading } from '../../components/ui';
import { useAgents } from './hooks';
import {
  AgentStatsBar,
  AgentGrid,
  AgentEmptyState,
  AgentRegisterModal,
  AgentEditModal,
  AgentMetricsModal,
} from './components';
import type { Agent } from '../../types';

export const AgentsPage: React.FC = () => {
  const { t } = useTranslation();
  const {
    agents,
    stats,
    loading,
    loadAgents,
    deleteAgent,
    updateAgent,
    syncConfig,
  } = useAgents();

  const [showRegisterModal, setShowRegisterModal] = useState(false);
  const [editingAgent, setEditingAgent] = useState<Agent | null>(null);
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null);

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(t('agents.actions.deleteConfirm', { name }))) {
      return;
    }
    await deleteAgent(id);
  };

  const handleSync = async (id: string) => {
    const success = await syncConfig(id);
    if (success) {
      console.log(t('agents.config_sync_triggered'));
    }
  };

  const handleUpdateAgent = async (id: string, name: string): Promise<boolean> => {
    return updateAgent(id, name);
  };

  if (loading && agents.length === 0) {
    return (
      <div className="row">
        <div className="col-12">
          <div className="card">
            <div className="card-body">
              <Loading text={t('common.loading')} />
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="row row-deck row-cards">
      <AgentStatsBar
        stats={stats}
        loading={loading}
        onRefresh={loadAgents}
        onRegister={() => setShowRegisterModal(true)}
      />

      {agents.length > 0 ? (
        <AgentGrid
          agents={agents}
          onSync={handleSync}
          onViewDetails={setSelectedAgent}
          onEdit={setEditingAgent}
          onDelete={handleDelete}
        />
      ) : (
        <AgentEmptyState onRegister={() => setShowRegisterModal(true)} />
      )}

      <AgentRegisterModal
        isOpen={showRegisterModal}
        onClose={() => setShowRegisterModal(false)}
      />

      <AgentEditModal
        agent={editingAgent}
        onClose={() => setEditingAgent(null)}
        onSave={handleUpdateAgent}
      />

      <AgentMetricsModal
        agent={selectedAgent}
        onClose={() => setSelectedAgent(null)}
      />
    </div>
  );
};

export default AgentsPage;
