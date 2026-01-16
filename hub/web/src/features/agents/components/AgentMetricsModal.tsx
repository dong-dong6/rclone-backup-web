import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconRefresh } from '@tabler/icons-react';
import {
  LineChart,
  Line,
  CartesianGrid,
  XAxis,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import { Modal, Loading } from '../../../components/ui';
import { useAgentMetrics } from '../hooks';
import { formatBytes } from '../../../utils/format';
import type { Agent } from '../../../types';

export interface AgentMetricsModalProps {
  agent: Agent | null;
  onClose: () => void;
}

const formatRate = (value: number) => `${formatBytes(value)}/s`;

export const AgentMetricsModal: React.FC<AgentMetricsModalProps> = ({
  agent,
  onClose,
}) => {
  const { t } = useTranslation();
  const { latest, chartData, loading } = useAgentMetrics(agent?.id || null);

  return (
    <Modal
      isOpen={!!agent}
      onClose={onClose}
      title={agent?.name || ''}
      size="lg"
      footer={
        <button onClick={onClose} className="btn btn-secondary">
          {t('common.close')}
        </button>
      }
    >
      {/* Real-time Monitoring */}
      <div className="mb-4">
        <div className="d-flex justify-content-between align-items-center mb-3">
          <h6 className="mb-0">实时监控</h6>
          {latest && (
            <small className="text-muted">
              更新于 {new Date(latest.recorded_at).toLocaleTimeString()}
            </small>
          )}
        </div>

        {loading ? (
          <div className="text-center py-5">
            <IconRefresh className="spinner text-primary" size={32} />
          </div>
        ) : (
          <div className="row g-3">
            {/* CPU */}
            <div className="col-md-6">
              <div className="border rounded p-3 h-100">
                <div className="d-flex justify-content-between small text-muted">
                  <span>CPU</span>
                  <span>{latest ? `${latest.cpu_usage.toFixed(1)}%` : '--'}</span>
                </div>
                <div className="progress progress-sm mt-2">
                  <div
                    className="progress-bar bg-primary"
                    role="progressbar"
                    style={{ width: `${latest?.cpu_usage ?? 0}%` }}
                  />
                </div>
              </div>
            </div>

            {/* Memory */}
            <div className="col-md-6">
              <div className="border rounded p-3 h-100">
                <div className="d-flex justify-content-between small text-muted">
                  <span>内存</span>
                  <span>{latest ? `${latest.memory_usage.toFixed(1)}%` : '--'}</span>
                </div>
                <p className="mb-1">
                  {latest
                    ? `${formatBytes(latest.memory_used)} / ${formatBytes(latest.memory_total)}`
                    : '--'}
                </p>
                <div className="progress progress-sm">
                  <div
                    className="progress-bar bg-info"
                    role="progressbar"
                    style={{ width: `${latest?.memory_usage ?? 0}%` }}
                  />
                </div>
              </div>
            </div>

            {/* Disk */}
            <div className="col-md-6">
              <div className="border rounded p-3 h-100">
                <div className="d-flex justify-content-between small text-muted">
                  <span>磁盘</span>
                  <span>{latest ? `${latest.disk_usage.toFixed(1)}%` : '--'}</span>
                </div>
                <p className="mb-1">
                  {latest
                    ? `${formatBytes(latest.disk_used)} / ${formatBytes(latest.disk_total)}`
                    : '--'}
                </p>
                <div className="progress progress-sm">
                  <div
                    className="progress-bar bg-warning"
                    role="progressbar"
                    style={{ width: `${latest?.disk_usage ?? 0}%` }}
                  />
                </div>
              </div>
            </div>

            {/* Network */}
            <div className="col-md-6">
              <div className="border rounded p-3 h-100">
                <div className="d-flex justify-content-between small text-muted">
                  <span>网络</span>
                  <span>
                    {latest
                      ? `${formatRate(latest.network_rx_rate)} ↓ / ${formatRate(latest.network_tx_rate)} ↑`
                      : '--'}
                  </span>
                </div>
                <p className="mb-1 text-muted">
                  TCP {latest?.tcp_connections ?? '--'} · UDP {latest?.udp_connections ?? '--'}
                </p>
                <p className="mb-0 text-muted">进程数 {latest?.process_count ?? '--'}</p>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* History Chart */}
      <div>
        <div className="d-flex justify-content-between align-items-center mb-3">
          <h6 className="mb-0">历史趋势 (6h)</h6>
        </div>

        {chartData.length > 0 ? (
          <ResponsiveContainer width="100%" height={240}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="time" />
              <Tooltip />
              <Legend />
              <Line
                type="monotone"
                dataKey="cpu"
                stroke="#0d6efd"
                strokeWidth={2}
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="memory"
                stroke="#0dcaf0"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <p className="text-muted mb-0">
            {loading ? '正在加载历史数据...' : '暂无历史监控数据。'}
          </p>
        )}
      </div>
    </Modal>
  );
};
