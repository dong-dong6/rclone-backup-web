import { useState, useEffect, useRef, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { remotesApi } from '../../../services';

type OAuthProvider = 'drive' | 'onedrive';

interface OAuthFlowState {
  flowId: string;
  provider: OAuthProvider;
  origin: string;
}

export function useOAuthFlow(
  onTokenReceived: (token: string) => void
) {
  const { t } = useTranslation();
  const [pending, setPending] = useState(false);

  const flowRef = useRef<OAuthFlowState | null>(null);
  const popupRef = useRef<Window | null>(null);
  const pollTimerRef = useRef<number | null>(null);
  const timeoutTimerRef = useRef<number | null>(null);
  const pollInFlightRef = useRef(false);

  const stop = useCallback(() => {
    flowRef.current = null;
    setPending(false);
    pollInFlightRef.current = false;

    if (pollTimerRef.current) {
      window.clearInterval(pollTimerRef.current);
      pollTimerRef.current = null;
    }
    if (timeoutTimerRef.current) {
      window.clearTimeout(timeoutTimerRef.current);
      timeoutTimerRef.current = null;
    }

    const popup = popupRef.current;
    if (popup && !popup.closed) {
      try {
        popup.close();
      } catch {
        // ignore
      }
    }
    popupRef.current = null;
  }, []);

  const fetchResult = useCallback(async (provider: OAuthProvider, flowId: string) => {
    if (pollInFlightRef.current) return;
    pollInFlightRef.current = true;

    try {
      const result = await remotesApi.getOAuthFlowStatus(provider, flowId);

      if (result.status === 'pending') return;

      if (result.status === 'success' && result.token) {
        onTokenReceived(JSON.stringify(result.token));
        stop();
        return;
      }

      stop();
      alert(t('remotes.oauth.failed', { message: result.error || '' }));
    } catch (error) {
      console.error('Failed to fetch OAuth result:', error);
    } finally {
      pollInFlightRef.current = false;
    }
  }, [onTokenReceived, stop, t]);

  // Handle postMessage from OAuth popup
  useEffect(() => {
    const handler = (event: MessageEvent) => {
      const data = event.data;
      if (!data || typeof data !== 'object') return;
      if (data.type !== 'rclone-oauth-result') return;

      const expected = flowRef.current;
      if (!expected || data.flow_id !== expected.flowId || data.provider !== expected.provider) return;
      if (event.origin !== expected.origin) return;

      if (data.ok) {
        void fetchResult(expected.provider, expected.flowId);
      } else {
        stop();
        alert(t('remotes.oauth.failed', { message: data.error || '' }));
      }
    };

    window.addEventListener('message', handler);
    return () => window.removeEventListener('message', handler);
  }, [fetchResult, stop, t]);

  const start = useCallback(async (
    provider: OAuthProvider,
    clientId: string,
    clientSecret: string,
    extraParams?: Record<string, string>
  ): Promise<void> => {
    if (pending) return;
    if (!clientId) {
      alert(t('remotes.oauth.missing_client_id'));
      return;
    }

    setPending(true);

    try {
      const result = await remotesApi.startOAuthFlow(provider, {
        client_id: clientId,
        client_secret: clientSecret,
        ...extraParams,
      });

      const origin = new URL(result.start_url, window.location.href).origin;
      flowRef.current = { flowId: result.flow_id, provider, origin };

      const popup = window.open(result.start_url, 'rclone-oauth', 'width=520,height=720');
      if (!popup) {
        stop();
        alert(t('remotes.oauth.popup_blocked'));
        return;
      }

      popupRef.current = popup;
      popup.focus();

      // Start polling
      if (pollTimerRef.current) {
        window.clearInterval(pollTimerRef.current);
      }
      pollTimerRef.current = window.setInterval(() => {
        const current = flowRef.current;
        if (!current) return;

        void fetchResult(current.provider, current.flowId);

        const w = popupRef.current;
        if (w && w.closed) {
          popupRef.current = null;
        }
      }, 1000);

      // Set timeout
      if (timeoutTimerRef.current) {
        window.clearTimeout(timeoutTimerRef.current);
      }
      timeoutTimerRef.current = window.setTimeout(() => {
        if (!flowRef.current) return;
        stop();
        alert(t('remotes.oauth.timeout'));
      }, 2 * 60 * 1000);
    } catch (error) {
      stop();
      console.error('Failed to start OAuth:', error);
      alert(t('remotes.oauth.start_failed'));
    }
  }, [pending, fetchResult, stop, t]);

  return {
    pending,
    start,
    stop,
  };
}
