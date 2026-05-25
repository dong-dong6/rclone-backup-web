import React from 'react';
import { useTranslation } from 'react-i18next';
import type { ExecutionFilter as FilterType } from '../hooks/useExecutions';

export interface ExecutionFilterProps {
  filter: FilterType;
  onFilterChange: (filter: Partial<FilterType>) => void;
}

export const ExecutionFilter: React.FC<ExecutionFilterProps> = ({
  filter,
  onFilterChange,
}) => {
  const { t } = useTranslation();

  return (
    <div className="row g-3 align-items-end">
      <div className="col-md-3">
        <label className="form-label">{t('executions.list.columns.status')}</label>
        <select
          value={filter.status}
          onChange={(e) => onFilterChange({ status: e.target.value })}
          className="form-select"
        >
          <option value="">{t('executions.filter.all_status')}</option>
          <option value="pending">{t('executions.status.pending')}</option>
          <option value="running">{t('executions.status.running')}</option>
          <option value="success">{t('executions.status.success')}</option>
          <option value="failed">{t('executions.status.failed')}</option>
        </select>
      </div>
    </div>
  );
};

export default ExecutionFilter;
