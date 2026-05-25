import React from 'react';
import { Col, Progress, Row, Statistic, Card, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import type { DashboardStats as DashboardStatsType } from '../hooks';

export interface DashboardStatsBarProps {
  stats: DashboardStatsType;
}

export const DashboardStatsBar: React.FC<DashboardStatsBarProps> = ({ stats }) => {
  const { t } = useTranslation();

  const cards = [
    {
      title: t('dashboard.stats.totalAgents'),
      value: stats.totalAgents,
      subtitle: `${t('dashboard.agents.online')}: ${stats.onlineAgents}`,
      percent: stats.totalAgents > 0 ? (stats.onlineAgents / stats.totalAgents) * 100 : 0,
      color: '#2563eb',
    },
    {
      title: t('dashboard.stats.activeTasks'),
      value: stats.activeTasks,
      subtitle: `${t('dashboard.stats.totalTasks')}: ${stats.totalTasks}`,
      percent: stats.totalTasks > 0 ? (stats.activeTasks / stats.totalTasks) * 100 : 0,
      color: '#16a34a',
    },
    {
      title: t('dashboard.stats.recentExecutions'),
      value: stats.recentExecutions,
      subtitle: t('dashboard.time_range.24h'),
    },
    {
      title: t('dashboard.stats.successRate'),
      value: `${stats.successRate.toFixed(1)}%`,
      percent: Math.round(stats.successRate),
      color: stats.successRate >= 90 ? '#16a34a' : stats.successRate >= 70 ? '#f59e0b' : '#dc2626',
    },
  ];

  return (
    <Row gutter={[16, 16]}>
      {cards.map((card) => (
        <Col xs={24} sm={12} lg={6} key={card.title}>
          <Card>
            <Statistic title={card.title} value={card.value} valueStyle={{ color: card.color }} />
            {card.subtitle && <Typography.Text type="secondary">{card.subtitle}</Typography.Text>}
            {typeof card.percent === 'number' && (
              <Progress percent={card.percent} showInfo={false} strokeColor={card.color} size="small" />
            )}
          </Card>
        </Col>
      ))}
    </Row>
  );
};
