import React, { useState, useEffect } from 'react';
import {
  IconServer,
  IconClock,
  IconCheck,
  IconX,
  IconRefresh,
  IconClockHour4,
} from '@tabler/icons-react';
import { LineChart, Line, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { useSSE } from '../contexts/SSEContext';
import api from '../services/api';
import './Dashboard.css';

// Removed Typography from antd

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
      success: { color: 'success', icon: <IconCheck size={16} /> },
      failed: { color: 'danger', icon: <IconX size={16} /> },
      running: { color: 'primary', icon: <IconRefresh size={16} className="spinner" /> },
      pending: { color: 'warning', icon: <IconClockHour4 size={16} /> },
    };

    const { color, icon } = config[status] || config.pending;
    return (
      <span className={`badge bg-${color} text-white`}>
        {icon}
        <span className="ms-1">{status.toUpperCase()}</span>
      </span>
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
    <div className="row row-deck row-cards">
      {/* Stats Cards */}
      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">Total Agents</div>
            </div>
            <div className="h1 mb-3">{stats.totalAgents}</div>
            <div className="d-flex mb-2">
              <div>Online: {stats.onlineAgents}</div>
            </div>
            <div className="progress progress-sm">
              <div 
                className="progress-bar bg-primary" 
                style={{ width: stats.totalAgents > 0 ? (stats.onlineAgents / stats.totalAgents) * 100 : 0 + '%' }}
              ></div>
            </div>
          </div>
        </div>
      </div>

      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">Active Tasks</div>
            </div>
            <div className="h1 mb-3 text-success">{stats.activeTasks}</div>
            <div className="d-flex mb-2">
              <div>Total: {stats.totalTasks}</div>
            </div>
            <div className="progress progress-sm">
              <div 
                className="progress-bar bg-success" 
                style={{ width: stats.totalTasks > 0 ? (stats.activeTasks / stats.totalTasks) * 100 : 0 + '%' }}
              ></div>
            </div>
          </div>
        </div>
      </div>

      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">Recent Executions</div>
            </div>
            <div className="h1 mb-3">{stats.recentExecutions}</div>
            <div className="d-flex mb-2">
              <div>Last 24h</div>
            </div>
          </div>
        </div>
      </div>

      <div className="col-sm-6 col-lg-3">
        <div className="card">
          <div className="card-body">
            <div className="d-flex align-items-center">
              <div className="subheader">Success Rate</div>
            </div>
            <div className="h1 mb-3">{stats.successRate.toFixed(1)}%</div>
            <div className="d-flex mb-2">
              <div className="progress progress-sm w-100">
                <div 
                  className={`progress-bar ${stats.successRate >= 90 ? 'bg-success' : stats.successRate >= 70 ? 'bg-warning' : 'bg-danger'}`}
                  style={{ width: Math.round(stats.successRate) + '%' }}
                ></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Charts Row */}
      <div className="col-12">
        <div className="row row-deck row-cards">
          <div className="col-12 col-lg-8">
            <div className="card">
              <div className="card-header">
                <h3 className="card-title">Backup Trend (24h)</h3>
              </div>
              <div className="card-body">
                <div style={{ height: '300px' }}>
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={backupTrend}>
                      <CartesianGrid strokeDasharray="3 3" stroke="rgba(0,0,0,0.1)" />
                      <XAxis dataKey="time" stroke="#6c757d" />
                      <YAxis stroke="#6c757d" />
                      <Tooltip />
                      <Area type="monotone" dataKey="success" stackId="1" stroke="#28a745" fill="#28a745" fillOpacity={0.6} />
                      <Area type="monotone" dataKey="failed" stackId="1" stroke="#dc3545" fill="#dc3545" fillOpacity={0.6} />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              </div>
            </div>
          </div>

          <div className="col-12 col-lg-4">
            <div className="card">
              <div className="card-header">
                <h3 className="card-title">Agent Status Distribution</h3>
              </div>
              <div className="card-body">
                <div className="mb-3">
                  <div className="d-flex align-items-center justify-content-between mb-2">
                    <span className="text-muted">Online Agents</span>
                    <span className="fw-bold">{stats.onlineAgents}/{stats.totalAgents}</span>
                  </div>
                  <div className="progress progress-sm">
                    <div 
                      className="progress-bar bg-success" 
                      style={{ width: stats.totalAgents > 0 ? (stats.onlineAgents / stats.totalAgents) * 100 : 0 + '%' }}
                    ></div>
                  </div>
                </div>

                <div className="mb-3">
                  <div className="d-flex align-items-center justify-content-between mb-2">
                    <span className="text-muted">Running Tasks</span>
                    <span className="fw-bold">30%</span>
                  </div>
                  <div className="progress progress-sm">
                    <div className="progress-bar bg-primary" style={{ width: '30%' }}></div>
                  </div>
                </div>

                <div className="mb-3">
                  <div className="d-flex align-items-center justify-content-between mb-2">
                    <span className="text-muted">Failed Tasks (24h)</span>
                    <span className="fw-bold">5%</span>
                  </div>
                  <div className="progress progress-sm">
                    <div className="progress-bar bg-danger" style={{ width: '5%' }}></div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Recent Executions Table */}
      <div className="col-12">
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">Recent Executions</h3>
          </div>
          <div className="card-body">
            <div className="table-responsive">
              <table className="table table-vcenter card-table">
                <thead>
                  <tr>
                    <th>Task</th>
                    <th>Agent</th>
                    <th>Status</th>
                    <th>Started</th>
                    <th>Duration</th>
                  </tr>
                </thead>
                <tbody>
                  {recentExecutions.map((execution) => (
                    <tr key={execution.id}>
                      <td>{execution.taskName}</td>
                      <td>{execution.agentName}</td>
                      <td>{getStatusTag(execution.status)}</td>
                      <td>{new Date(execution.startedAt).toLocaleString()}</td>
                      <td>{Math.round(execution.duration)}s</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;