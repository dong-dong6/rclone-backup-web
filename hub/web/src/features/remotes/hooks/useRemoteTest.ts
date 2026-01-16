import { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { remotesApi } from '../../../services';
import type { RcloneRemote, RemoteTestResponse } from '../../../types';

export function useRemoteTest(onTestComplete: () => void) {
  const { t } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const [remote, setRemote] = useState<RcloneRemote | null>(null);
  const [testPath, setTestPath] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<RemoteTestResponse | null>(null);

  const open = useCallback((targetRemote: RcloneRemote) => {
    setRemote(targetRemote);
    setTestPath('');
    setResult(null);
    setIsOpen(true);
  }, []);

  const close = useCallback(() => {
    if (submitting) return;
    setIsOpen(false);
    setRemote(null);
    setResult(null);
    setTestPath('');
  }, [submitting]);

  const runTest = useCallback(async () => {
    if (!remote) return;
    if (submitting) return;

    setSubmitting(true);
    setResult(null);

    try {
      const response = await remotesApi.test(remote.id, testPath.trim() || undefined);
      setResult(response);
    } catch (error: any) {
      const data = error?.response?.data;
      if (data && typeof data === 'object') {
        setResult(data as RemoteTestResponse);
      } else {
        setResult({
          success: false,
          message: t('errors.server'),
          error: String(error),
        });
      }
    } finally {
      setSubmitting(false);
      onTestComplete();
    }
  }, [remote, testPath, submitting, t, onTestComplete]);

  return {
    isOpen,
    remote,
    testPath,
    submitting,
    result,
    setTestPath,
    open,
    close,
    runTest,
  };
}
