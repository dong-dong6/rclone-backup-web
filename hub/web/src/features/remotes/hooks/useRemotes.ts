import { useState, useEffect, useCallback } from 'react';
import { remotesApi } from '../../../services';
import type { RcloneRemote } from '../../../types';

export function useRemotes() {
  const [remotes, setRemotes] = useState<RcloneRemote[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchRemotes = useCallback(async () => {
    setLoading(true);
    try {
      const data = await remotesApi.getAll();
      setRemotes(data);
    } catch (error) {
      console.error('Failed to fetch remotes:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  const deleteRemote = useCallback(async (id: string): Promise<boolean> => {
    try {
      await remotesApi.delete(id);
      setRemotes(prev => prev.filter(r => r.id !== id));
      return true;
    } catch (error) {
      console.error('Failed to delete remote:', error);
      return false;
    }
  }, []);

  useEffect(() => {
    fetchRemotes();
  }, [fetchRemotes]);

  return {
    remotes,
    loading,
    fetchRemotes,
    deleteRemote,
  };
}
