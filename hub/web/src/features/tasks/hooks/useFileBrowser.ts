import { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { agentsApi } from '../../../services';
import type { FSListEntry } from '../../../types';

export function useFileBrowser() {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [agentId, setAgentId] = useState<string | null>(null);
  const [path, setPath] = useState('');
  const [parent, setParent] = useState('');
  const [entries, setEntries] = useState<FSListEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchDirectory = useCallback(async (targetAgentId: string, targetPath: string) => {
    setLoading(true);
    setError(null);

    try {
      const response = await agentsApi.listDirectory(targetAgentId, targetPath);
      setPath(response.path || targetPath);
      setParent(response.parent || '');
      setEntries(response.entries || []);
    } catch (err: any) {
      const message = err?.response?.data?.message || err?.response?.data?.error || t('errors.server');
      setError(message);
      setEntries([]);
      setParent('');
    } finally {
      setLoading(false);
    }
  }, [t]);

  const open = useCallback((selectedAgentId: string | null, initialPath: string) => {
    setIsOpen(true);
    setError(null);
    setEntries([]);
    setParent('');

    if (!selectedAgentId) {
      setAgentId(null);
      setPath(initialPath || '/');
      setError(t('tasks.browse.select_agent_first'));
      return;
    }

    setAgentId(selectedAgentId);
    const startPath = (initialPath || '/').trim() || '/';
    setPath(startPath);
    void fetchDirectory(selectedAgentId, startPath);
  }, [fetchDirectory, t]);

  const close = useCallback(() => {
    if (loading) return;
    setIsOpen(false);
    setAgentId(null);
    setPath('');
    setParent('');
    setEntries([]);
    setError(null);
  }, [loading]);

  const navigate = useCallback((targetPath: string) => {
    if (!agentId || loading) return;
    const next = targetPath.trim();
    if (!next) return;
    setPath(next);
    void fetchDirectory(agentId, next);
  }, [agentId, loading, fetchDirectory]);

  return {
    isOpen,
    agentId,
    path,
    parent,
    entries,
    loading,
    error,
    setPath,
    open,
    close,
    navigate,
  };
}
