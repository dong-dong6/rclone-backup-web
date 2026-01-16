import { useState, useEffect, useRef } from 'react';
import { useSSE } from '../../../contexts/SSEContext';
import { agentsApi } from '../../../services';
import type { AgentMetric } from '../../../types';

export function useAgentMetrics(agentId: string | null) {
  const { subscribe } = useSSE();
  const [latest, setLatest] = useState<AgentMetric | null>(null);
  const [history, setHistory] = useState<AgentMetric[]>([]);
  const [loading, setLoading] = useState(false);

  // Use ref to track agentId for SSE handler without re-subscribing
  const agentIdRef = useRef<string | null>(null);
  useEffect(() => {
    agentIdRef.current = agentId;
  }, [agentId]);

  useEffect(() => {
    if (!agentId) {
      setLatest(null);
      setHistory([]);
      return;
    }

    let cancelled = false;

    const loadMetrics = async () => {
      setLoading(true);
      try {
        const [latestData, historyData] = await Promise.all([
          agentsApi.getMetricsLatest(agentId),
          agentsApi.getMetricsHistory(agentId, 6),
        ]);

        if (cancelled) return;

        setLatest(latestData);
        setHistory(historyData);
      } catch (error) {
        if (!cancelled) {
          console.error('Failed to load metrics:', error);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    loadMetrics();

    return () => {
      cancelled = true;
    };
  }, [agentId]);

  // Real-time metrics updates via SSE
  useEffect(() => {
    const unsubscribe = subscribe('agent.heartbeat', (event: {
      data: {
        agent_id: string;
        metrics?: AgentMetric;
      }
    }) => {
      const { agent_id, metrics } = event.data;
      const currentAgentId = agentIdRef.current;

      if (currentAgentId && agent_id === currentAgentId && metrics) {
        const newMetric: AgentMetric = {
          ...metrics,
          recorded_at: metrics.recorded_at || new Date().toISOString(),
        };

        setLatest(newMetric);
        setHistory(prev => {
          const updated = [...prev, newMetric];
          return updated.slice(-360); // Keep last 360 data points
        });
      }
    });

    return unsubscribe;
  }, [subscribe]);

  const chartData = history.map((metric) => ({
    time: new Date(metric.recorded_at).toLocaleTimeString(),
    cpu: Number(metric.cpu_usage.toFixed(2)),
    memory: Number(metric.memory_usage.toFixed(2)),
  }));

  return {
    latest,
    history,
    chartData,
    loading,
  };
}
