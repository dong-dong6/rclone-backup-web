import React from 'react';
import { Card, Table } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useTranslation } from 'react-i18next';
import { StatusBadge } from '../../../components/ui';
import type { RecentExecution } from '../hooks';
import type { StatusType } from '../../../components/ui';

export interface RecentExecutionsTableProps {
  executions: RecentExecution[];
}

export const RecentExecutionsTable: React.FC<RecentExecutionsTableProps> = ({
  executions,
}) => {
  const { t } = useTranslation();

  const columns: ColumnsType<RecentExecution> = [
    {
      title: t('executions.list.columns.task'),
      dataIndex: 'taskName',
    },
    {
      title: t('executions.list.columns.agent'),
      dataIndex: 'agentName',
    },
    {
      title: t('executions.list.columns.status'),
      dataIndex: 'status',
      render: (status: RecentExecution['status']) => (
        <StatusBadge status={status as StatusType} label={t(`executions.status.${status}`)} />
      ),
    },
    {
      title: t('executions.list.columns.startedAt'),
      dataIndex: 'startedAt',
      render: (startedAt?: string) => (startedAt ? new Date(startedAt).toLocaleString() : '-'),
    },
    {
      title: t('executions.list.columns.duration'),
      dataIndex: 'duration',
      render: (duration: number) => `${Math.round(duration)}s`,
    },
  ];

  return (
    <Card title={t('dashboard.recent_executions.title')}>
      <Table
        rowKey="id"
        columns={columns}
        dataSource={executions}
        pagination={false}
        locale={{ emptyText: t('dashboard.recent_executions.no_executions') }}
      />
    </Card>
  );
};
