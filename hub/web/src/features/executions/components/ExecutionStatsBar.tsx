import React from 'react';
import { useTranslation } from 'react-i18next';
import { StatsCard } from '../../../components/ui';
import { formatDuration } from '../../../utils';
import type { ExecutionStats } from '../../../types';

export interface ExecutionStatsBarProps {
  stats: ExecutionStats | null;
}

export const ExecutionStatsBar: React.FC<ExecutionStatsBarProps> = ({ stats }) => {
  const { t } = useTranslation();

  if (!stats) return null;

  return (
    <>
      <div className="col-sm-6 col-lg-2">
        <StatsCard
          title={t('executions.stats.total')}
          value={stats.total}
        />
      </div>
      <div className="col-sm-6 col-lg-2">
        <StatsCard
          title={t('executions.stats.success')}
          value={stats.success}
          valueColor="text-success"
        />
      </div>
      <div className="col-sm-6 col-lg-2">
        <StatsCard
          title={t('executions.stats.failed')}
          value={stats.failed}
          valueColor="text-danger"
        />
      </div>
      <div className="col-sm-6 col-lg-2">
        <StatsCard
          title={t('executions.stats.running')}
          value={stats.running}
          valueColor="text-primary"
        />
      </div>
      <div className="col-sm-6 col-lg-2">
        <StatsCard
          title={t('executions.stats.success_rate')}
          value={`${stats.success_rate_24h.toFixed(1)}%`}
        />
      </div>
      <div className="col-sm-6 col-lg-2">
        <StatsCard
          title={t('executions.stats.avg_duration')}
          value={formatDuration(stats.avg_duration_seconds)}
        />
      </div>
    </>
  );
};

export default ExecutionStatsBar;
