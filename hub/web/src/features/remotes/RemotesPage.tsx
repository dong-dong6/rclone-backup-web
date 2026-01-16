import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { IconRefresh, IconPlus } from '@tabler/icons-react';
import { Loading } from '../../components/ui';
import { useRemotes, useRemoteForm, useOAuthFlow, useRemoteTest } from './hooks';
import { RemoteCard, RemoteTestModal, RemoteModal, RemoteEmptyState } from './components';
import { normalizeTokenJson } from './utils';

export const RemotesPage: React.FC = () => {
  const { t } = useTranslation();
  const { remotes, loading, fetchRemotes, deleteRemote } = useRemotes();
  const [testingRemoteId, setTestingRemoteId] = useState<string | null>(null);

  const form = useRemoteForm(fetchRemotes);

  const oauth = useOAuthFlow((token) => {
    form.updateGuidedValue('token', normalizeTokenJson(token));
  });

  const test = useRemoteTest(fetchRemotes);

  const handleDelete = async (id: string) => {
    const remote = remotes.find(r => r.id === id);
    if (!confirm(t('remotes.actions.deleteConfirm', { name: remote?.name || '' }))) {
      return;
    }
    await deleteRemote(id);
  };

  const handleTest = (id: string) => {
    const remote = remotes.find(r => r.id === id);
    if (remote) {
      setTestingRemoteId(id);
      test.open(remote);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    oauth.stop();
    await form.submit();
  };

  const handleStartOAuth = (provider: 'drive' | 'onedrive') => {
    const clientId = (form.guidedValues.client_id || '').trim();
    const clientSecret = (form.guidedValues.client_secret || '').trim();

    const extraParams: Record<string, string> = {};
    if (provider === 'drive') {
      extraParams.scope = (form.guidedValues.scope || '').trim();
    } else {
      extraParams.region = (form.guidedValues.region || '').trim() || 'global';
    }

    oauth.start(provider, clientId, clientSecret, extraParams);
  };

  const handleCloseForm = () => {
    oauth.stop();
    form.close();
  };

  const handleCloseTest = () => {
    test.close();
    setTestingRemoteId(null);
  };

  if (loading && remotes.length === 0) {
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
      <div className="col-12">
        <div className="card">
          <div className="card-body d-flex justify-content-end gap-2">
            <button onClick={fetchRemotes} className="btn btn-outline-primary" disabled={loading}>
              <IconRefresh size={16} className={loading ? 'spinner' : undefined} />
              <span className="ms-1">{t('common.refresh')}</span>
            </button>
            <button onClick={form.openCreate} className="btn btn-primary">
              <IconPlus size={16} />
              <span className="ms-1">{t('remotes.create.title')}</span>
            </button>
          </div>
        </div>
      </div>

      {remotes.length > 0 ? (
        remotes.map((remote) => (
          <RemoteCard
            key={remote.id}
            remote={remote}
            testing={testingRemoteId === remote.id && test.submitting}
            onTest={handleTest}
            onEdit={form.openEdit}
            onDelete={handleDelete}
          />
        ))
      ) : (
        <RemoteEmptyState onCreate={form.openCreate} />
      )}

      <RemoteTestModal
        isOpen={test.isOpen}
        remote={test.remote}
        testPath={test.testPath}
        submitting={test.submitting}
        result={test.result}
        onPathChange={test.setTestPath}
        onClose={handleCloseTest}
        onTest={test.runTest}
      />

      <RemoteModal
        isOpen={form.isOpen}
        isEditing={!!form.editingRemote}
        modalLoading={form.modalLoading}
        configMode={form.configMode}
        presetKey={form.presetKey}
        guidedValues={form.guidedValues}
        formData={form.formData}
        currentPreset={form.currentPreset}
        presets={form.presets}
        previewConfig={form.previewConfig}
        oauthPending={oauth.pending}
        onClose={handleCloseForm}
        onSubmit={handleSubmit}
        onSwitchToGuided={form.switchToGuided}
        onSwitchToRaw={form.switchToRaw}
        onPresetChange={form.changePreset}
        onGuidedValueChange={form.updateGuidedValue}
        onFormDataChange={form.updateFormData}
        onStartOAuth={handleStartOAuth}
      />
    </div>
  );
};

export default RemotesPage;
