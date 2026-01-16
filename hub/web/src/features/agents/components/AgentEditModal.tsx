import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal } from '../../../components/ui';
import type { Agent } from '../../../types';

export interface AgentEditModalProps {
  agent: Agent | null;
  onClose: () => void;
  onSave: (id: string, name: string) => Promise<boolean>;
}

export const AgentEditModal: React.FC<AgentEditModalProps> = ({
  agent,
  onClose,
  onSave,
}) => {
  const { t } = useTranslation();
  const [name, setName] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (agent) {
      setName(agent.name);
    }
  }, [agent]);

  const handleSave = async () => {
    if (!agent || !name.trim()) return;

    setSaving(true);
    const success = await onSave(agent.id, name.trim());
    setSaving(false);

    if (success) {
      onClose();
    }
  };

  const handleClose = () => {
    setName('');
    onClose();
  };

  const canSave = agent && name.trim() && name !== agent.name;

  return (
    <Modal
      isOpen={!!agent}
      onClose={handleClose}
      title={t('agents.edit.title')}
      footer={
        <>
          <button onClick={handleClose} className="btn btn-secondary">
            {t('common.cancel')}
          </button>
          <button
            onClick={handleSave}
            className="btn btn-primary"
            disabled={!canSave || saving}
          >
            {saving ? t('common.saving') : t('common.save')}
          </button>
        </>
      }
    >
      <div className="mb-3">
        <label className="form-label">{t('agents.edit.name_label')}</label>
        <input
          type="text"
          className="form-control"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('agents.edit.name_placeholder')}
        />
      </div>
    </Modal>
  );
};
