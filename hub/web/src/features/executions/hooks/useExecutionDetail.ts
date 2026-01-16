import { useState, useEffect, useCallback, useRef } from 'react';
import { executionsApi } from '../../../services';
import { useSSE } from '../../../contexts/SSEContext';
import type { TaskExecution } from '../../../types';

export interface UseExecutionDetailReturn {
  execution: TaskExecution | null;
  logs: string;
  loading: boolean;
  loadError: boolean;
  autoScroll: boolean;
  setAutoScroll: (value: boolean) => void;
  refresh: () => Promise<void>;
  downloadLogs: () => void;
  logsContainerRef: React.RefObject<HTMLDivElement>;
  logsEndRef: React.RefObject<HTMLDivElement>;
  handleScroll: () => void;
}

export function useExecutionDetail(id: string | undefined): UseExecutionDetailReturn {
  const { subscribe } = useSSE();
  const [execution, setExecution] = useState<TaskExecution | null>(null);
  const [logs, setLogs] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [autoScroll, setAutoScroll] = useState(true);
  const logsEndRef = useRef<HTMLDivElement>(null);
  const logsContainerRef = useRef<HTMLDivElement>(null);

  const fetchExecutionDetail = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setLoadError(false);

    try {
      const response = await executionsApi.getById(id);
      setExecution(response.data);
      setLogs(response.data?.log_output || '');
    } catch (error) {
      console.error('Failed to fetch execution detail:', error);
      setExecution(null);
      setLogs('');
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    if (!id) return;
    fetchExecutionDetail();
  }, [id, fetchExecutionDetail]);

  // SSE subscriptions
  useEffect(() => {
    if (!id) return;

    const unsubscribeLog = subscribe('execution.log.update', (event) => {
      if (event.data?.execution_id !== id) return;
      const timestamp = event.data?.log?.timestamp;
      const message = event.data?.log?.message;
      if (!message) return;
      const line = timestamp ? `[${timestamp}] ${message}` : message;
      setLogs((prev) => prev + line + '\n');
    });

    const unsubscribeStatus = subscribe('execution.status.update', (event) => {
      if (event.data?.execution_id !== id) return;
      const status = event.data?.status as TaskExecution['status'] | undefined;
      if (!status) return;
      const errorMessage = event.data?.error_message as string | undefined;
      setExecution((prev) =>
        prev
          ? {
              ...prev,
              status,
              ...(errorMessage ? { error_message: errorMessage } : {}),
            }
          : prev
      );
    });

    return () => {
      unsubscribeLog();
      unsubscribeStatus();
    };
  }, [id, subscribe]);

  // Auto scroll
  useEffect(() => {
    if (autoScroll && logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [logs, autoScroll]);

  const handleScroll = useCallback(() => {
    if (!logsContainerRef.current) return;
    const { scrollTop, scrollHeight, clientHeight } = logsContainerRef.current;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 10;
    setAutoScroll(isAtBottom);
  }, []);

  const downloadLogs = useCallback(() => {
    const blob = new Blob([logs], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `execution-${id}-logs.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }, [logs, id]);

  return {
    execution,
    logs,
    loading,
    loadError,
    autoScroll,
    setAutoScroll,
    refresh: fetchExecutionDetail,
    downloadLogs,
    logsContainerRef,
    logsEndRef,
    handleScroll,
  };
}

export default useExecutionDetail;
