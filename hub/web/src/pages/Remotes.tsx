import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Plus, Edit, Trash2 } from 'lucide-react';
import { apiClient } from '../services/api';
import { useAuth } from '../contexts/AuthContext';

interface RcloneRemote {
  id: string;
  name: string;
  config_data: string;
  created_at: string;
}

const Remotes: React.FC = () => {
  const { t } = useTranslation();
  const { token } = useAuth();
  const [remotes, setRemotes] = useState<RcloneRemote[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingRemote, setEditingRemote] = useState<RcloneRemote | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    config_data: '',
  });

  useEffect(() => {
    fetchRemotes();
  }, []);

  const fetchRemotes = async () => {
    setLoading(true);
    try {
      const response = await apiClient.get('/admin/remotes', {
        headers: { Authorization: `Bearer ${token}` },
      });
      setRemotes(response.data);
    } catch (error) {
      console.error('Failed to fetch remotes:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateRemote = () => {
    setEditingRemote(null);
    setFormData({
      name: '',
      config_data: '',
    });
    setShowCreateModal(true);
  };

  const handleEditRemote = (remote: RcloneRemote) => {
    setEditingRemote(remote);
    setFormData({
      name: remote.name,
      config_data: remote.config_data,
    });
    setShowCreateModal(true);
  };

  const handleDeleteRemote = async (remoteId: string) => {
    if (!confirm(t('remotes.confirm_delete'))) return;

    try {
      await apiClient.delete(`/admin/remotes/${remoteId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      fetchRemotes();
    } catch (error) {
      console.error('Failed to delete remote:', error);
      alert(t('remotes.delete_failed'));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      if (editingRemote) {
        await apiClient.put(`/admin/remotes/${editingRemote.id}`, formData, {
          headers: { Authorization: `Bearer ${token}` },
        });
      } else {
        await apiClient.post('/admin/remotes', formData, {
          headers: { Authorization: `Bearer ${token}` },
        });
      }

      setShowCreateModal(false);
      fetchRemotes();
    } catch (error) {
      console.error('Failed to save remote:', error);
      alert(t('remotes.save_failed'));
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="neu-card p-8">
          <div className="animate-spin rounded-full h-12 w-12 border-4 border-primary border-t-transparent"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="page-container">
      <div className="page-header">
        <h1 className="page-title">{t('remotes.title')}</h1>
        <button
          onClick={handleCreateRemote}
          className="neu-button-primary"
        >
          <Plus size={20} />
          <span>{t('remotes.create_new')}</span>
        </button>
      </div>

      <div className="tasks-grid">
        {remotes.map((remote) => (
          <div key={remote.id} className="neu-card task-card">
            <div className="task-card-header">
              <h3 className="text-xl font-semibold mb-1">{remote.name}</h3>
              <div className="flex space-x-1">
                <button
                  onClick={() => handleEditRemote(remote)}
                  className="neu-button-icon"
                  title={t('common.edit')}
                >
                  <Edit size={16} />
                </button>
                <button
                  onClick={() => handleDeleteRemote(remote.id)}
                  className="neu-button-icon text-red-500"
                  title={t('common.delete')}
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>

      {remotes.length === 0 && (
        <div className="neu-card p-12 text-center">
          <p className="text-gray-500 dark:text-gray-400 mb-4">
            {t('remotes.no_remotes')}
          </p>
          <button
            onClick={handleCreateRemote}
            className="neu-button-primary"
          >
            {t('remotes.create_first')}
          </button>
        </div>
      )}

      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="neu-card p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <h2 className="text-2xl font-bold mb-6">
              {editingRemote ? t('remotes.edit_remote') : t('remotes.create_remote')}
            </h2>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('remotes.remote_name')}
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  className="neu-input w-full"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">
                  {t('remotes.config_data')}
                </label>
                <textarea
                  value={formData.config_data}
                  onChange={(e) => setFormData({ ...formData, config_data: e.target.value })}
                  className="neu-input w-full"
                  rows={10}
                  required
                />
              </div>
              <div className="flex justify-end space-x-3 pt-4">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="neu-button"
                >
                  {t('common.cancel')}
                </button>
                <button type="submit" className="neu-button-primary">
                  {editingRemote ? t('common.save') : t('common.create')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default Remotes;
