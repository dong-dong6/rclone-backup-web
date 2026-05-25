import React from 'react';
import { Button, Card, Col, Row, Space, Statistic } from 'antd';
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { AgentStats } from '../../../types';

export interface AgentStatsBarProps {
  stats: AgentStats;
  loading?: boolean;
  onRefresh: () => void;
  onRegister: () => void;
}

export const AgentStatsBar: React.FC<AgentStatsBarProps> = ({
  stats,
  loading,
  onRefresh,
  onRegister,
}) => {
  const { t } = useTranslation();

  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} sm={12} lg={6}>
        <Card><Statistic title={t('agents.stats.total')} value={stats.total} /></Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card><Statistic title={t('agents.stats.online')} value={stats.online} valueStyle={{ color: '#16a34a' }} /></Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card><Statistic title={t('agents.stats.running')} value={stats.running} valueStyle={{ color: '#2563eb' }} /></Card>
      </Col>
      <Col xs={24} sm={12} lg={6}>
        <Card>
          <Statistic title={t('common.actions')} value=" " />
          <Space wrap>
            <Button icon={<ReloadOutlined />} onClick={onRefresh} loading={loading}>
              {t('common.refresh')}
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={onRegister}>
              {t('agents.register.new')}
            </Button>
          </Space>
        </Card>
      </Col>
    </Row>
  );
};
