import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Progress, Table, Tag, Space, Typography } from 'antd';
import {
  CloudServerOutlined,
  ScheduleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  SyncOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons';
import { LineChart, Line, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { useSSE } from '../contexts/SSEContext';
import api from '../services/api';

const { Title, Text } = Typography;

interface DashboardStats {
  totalAgents: number;
  onlineAgents: number;
  totalTasks: number;
  activeTasks: number;
  recentExecutions: number;
  successRate: number;
}

interface RecentExecution {
  id: string;
  taskName: string;
  agentName: string;
  status: string;
  startedAt: string;
  duration: number;
}

const Dashboard: React.FC = () => {
  const [stats, setStats] = useState<DashboardStats>({
    totalAgents: 0,
    onlineAgents: 0,
    totalTasks: 0,
    activeTasks: 0,
    recentExecutions: 0,
    successRate: 0,
  });
  const [recentExecutions, setRecentExecutions] = useState<RecentExecution[]>([]);
  const [loading, setLoading] = useState(true);
  const { events } = useSSE();

  useEffect(() => {
    fetchDashboardData();
    const interval = setInterval(fetchDashboardData, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    // Handle SSE events
    if (events.length > 0) {
      const latestEvent = events[events.length - 1];
      if (latestEvent.type === 'agent.status.update' || 
          latestEvent.type === 'execution.status.update') {
        fetchDashboardData();
      }
    }
  }, [events]);

  const fetchDashboardData = async () => {
    try {
      setLoading(true);
      const [agentsRes, tasksRes, executionsRes] = await Promise.all([
        api.get('/admin/agents'),
        api.get('/admin/tasks'),
        api.get('/admin/executions?limit=10'),
      ]);

      const agents = agentsRes.data;
      const tasks = tasksRes.data;
      const executions = executionsRes.data.items || [];

      const onlineAgents = agents.filter((a: any) => a.status === 'online').length;
      const activeTasks = tasks.filter((t: any) => t.is_active).length;
      
      const successfulExecutions = executions.filter((e: any) => e.status === 'success').length;
      const successRate = executions.length > 0 
        ? (successfulExecutions / executions.length) * 100 
        : 0;

      setStats({
        totalAgents: agents.length,
        onlineAgents,
        totalTasks: tasks.length,
        activeTasks,
        recentExecutions: executions.length,
        successRate,
      });

      setRecentExecutions(executions.slice(0, 5).map((e: any) => ({
        id: e.id,
        taskName: e.task_name,
        agentName: e.agent_name,
        status: e.status,
        startedAt: e.started_at,
        duration: e.duration_seconds || 0,
      })));
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusTag = (status: string) => {
    const config: Record<string, { color: string; icon: React.ReactNode }> = {
      success: { color: 'success', icon: <CheckCircleOutlined /> },
      failed: { color: 'error', icon: <CloseCircleOutlined /> },
      running: { color: 'processing', icon: <SyncOutlined spin /> },
      pending: { color: 'default', icon: <ClockCircleOutlined /> },
    };

    const { color, icon } = config[status] || config.pending;
    return (
      <Tag color={color} icon={icon}>
        {status.toUpperCase()}
      </Tag>
    );
  };

  const columns = [
    {
      title: 'Task',
      dataIndex: 'taskName',
      key: 'taskName',
    },
    {
      title: 'Agent',
      dataIndex: 'agentName',
      key: 'agentName',
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => getStatusTag(status),
    },
    {
      title: 'Started',
      dataIndex: 'startedAt',
      key: 'startedAt',
      render: (time: string) => new Date(time).toLocaleString(),
    },
    {
      title: 'Duration',
      dataIndex: 'duration',
      key: 'duration',
      render: (duration: number) => `${Math.round(duration)}s`,
    },
  ];

  // Mock data for charts
  const backupTrend = [
    { time: '00:00', success: 45, failed: 2 },
    { time: '04:00', success: 52, failed: 1 },
    { time: '08:00', success: 61, failed: 3 },
    { time: '12:00', success: 74, failed: 2 },
    { time: '16:00', success: 89, failed: 4 },
    { time: '20:00', success: 95, failed: 2 },
  ];

  return (
    <div>
      <Title level={2}>Dashboard</Title>
      
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Total Agents"
              value={stats.totalAgents}
              prefix={<CloudServerOutlined />}
              suffix={
                <Text type="secondary">
                  ({stats.onlineAgents} online)
                </Text>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Active Tasks"
              value={stats.activeTasks}
              prefix={<ScheduleOutlined />}
              suffix={
                <Text type="secondary">
                  / {stats.totalTasks}
                </Text>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Recent Executions"
              value={stats.recentExecutions}
              prefix={<SyncOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Statistic
                title="Success Rate"
                value={stats.successRate}
                precision={1}
                suffix="%"
              />
              <Progress
                type="circle"
                percent={Math.round(stats.successRate)}
                width={60}
                status={stats.successRate >= 90 ? 'success' : stats.successRate >= 70 ? 'normal' : 'exception'}
              />
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={16}>
          <Card title="Backup Trend (24h)">
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={backupTrend}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="time" />
                <YAxis />
                <Tooltip />
                <Area type="monotone" dataKey="success" stackId="1" stroke="#52c41a" fill="#52c41a" />
                <Area type="monotone" dataKey="failed" stackId="1" stroke="#ff4d4f" fill="#ff4d4f" />
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="Agent Status Distribution">
            <div style={{ padding: '20px' }}>
              <Space direction="vertical" style={{ width: '100%' }}>
                <div>
                  <Text>Online Agents</Text>
                  <Progress
                    percent={(stats.onlineAgents / stats.totalAgents) * 100}
                    status="active"
                    strokeColor="#52c41a"
                  />
                </div>
                <div>
                  <Text>Running Tasks</Text>
                  <Progress
                    percent={30}
                    status="active"
                    strokeColor="#1890ff"
                  />
                </div>
                <div>
                  <Text>Failed Tasks (24h)</Text>
                  <Progress
                    percent={5}
                    status="exception"
                  />
                </div>
              </Space>
            </div>
          </Card>
        </Col>
      </Row>

      <Card title="Recent Executions">
        <Table
          columns={columns}
          dataSource={recentExecutions}
          rowKey="id"
          loading={loading}
          pagination={false}
        />
      </Card>
    </div>
  );
};

export default Dashboard;