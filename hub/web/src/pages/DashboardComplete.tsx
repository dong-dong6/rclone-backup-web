import React, { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { 
  Server, Activity, CheckCircle, XCircle, Clock, 
  TrendingUp, Calendar, Database, Zap, AlertTriangle,
  RefreshCw, ChevronRight, ArrowUp, ArrowDown
} from 'lucide-react';
import { 
  LineChart, Line, AreaChart, Area, BarChart, Bar, 
  PieChart, Pie, Cell, XAxis, YAxis, CartesianGrid, 
  Tooltip, Legend, ResponsiveContainer 
} from 'recharts';
import { apiService } from '../services/api';
import { useAuth } from '../contexts/AuthContext';
import { useSSE } from '../contexts/SSEContext';
import classNames from 'classnames';
import '../styles/dashboard.css';

// Types
interface DashboardStats {
  agents: {
    total: number;
    online: number;
    offline: number;
    running_task: number;
  };
  tasks: {
    total: number;
    active: number;
    scheduled_today: number;
  };
  executions: {
    total_24h: number;
    success_24h: number;
    failed_24h: number;
    running: number;
    success_rate: number;
    avg_duration: number;
  };
  storage: {
    total_backup_size: number;
    daily_growth: number;
    most_used_remote: string;
  };
}

interface RecentExecution {
  id: string;
  task_name: string;
  agent_name: string;
  status: 'pending' | 'running' | 'success' | 'failed';
  started_at: string;
  duration?: number;
}

interface SystemHealth {
  hub_status: 'healthy' | 'degraded' | 'down';
  database_status: 'healthy' | 'degraded' | 'down';
  avg_agent_response_time: number;
  system_load: number;
}

const DashboardComplete: React.FC = () => {
  const { t } = useTranslation();
  const { token } = useAuth();
  const { subscribe } = useSSE();
  
  // State
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [recentExecutions, setRecentExecutions] = useState<RecentExecution[]>([]);
  const [systemHealth, setSystemHealth] = useState<SystemHealth | null>(null);
  const [chartData, setChartData] = useState<any[]>([]);
  const [refreshing, setRefreshing] = useState(false);
  const [timeRange, setTimeRange] = useState('24h'); // 24h, 7d, 30d

  // Load dashboard data
  useEffect(() => {
    loadDashboardData();
    
    // Setup SSE subscriptions for real-time updates
    const unsubscribers = [
      subscribe('agent.status.update', handleAgentUpdate),
      subscribe('execution.status.update', handleExecutionUpdate),
      subscribe('task.dispatched', handleTaskDispatched),
    ];
    
    // Refresh data every 30 seconds
    const interval = setInterval(() => {
      refreshDashboardData();
    }, 30000);
    
    return () => {
      unsubscribers.forEach(unsub => unsub());
      clearInterval(interval);
    };
  }, [timeRange]);

  const loadDashboardData = async () => {
    setLoading(true);
    try {
      const [statsRes, executionsRes, healthRes, chartRes] = await Promise.all([
        apiService.getDashboardStats(),
        apiService.getRecentActivity(),
        apiService.getSystemHealth(),
        apiService.getChartData(timeRange),
      ]);
      
      setStats(statsRes.data);
      setRecentExecutions(executionsRes.data);
      setSystemHealth(healthRes.data);
      setChartData(chartRes.data);
    } catch (error) {
      console.error('Failed to load dashboard:', error);
    } finally {
      setLoading(false);
    }
  };

  const refreshDashboardData = async () => {
    if (refreshing) return;
    
    setRefreshing(true);
    try {
      const statsRes = await apiService.getDashboardStats();
      setStats(statsRes.data);
      
      const executionsRes = await apiService.getRecentActivity();
      setRecentExecutions(executionsRes.data);
    } catch (error) {
      console.error('Failed to refresh dashboard:', error);
    } finally {
      setRefreshing(false);
    }
  };

  // SSE Event Handlers
  const handleAgentUpdate = (event: any) => {
    // Update agent stats in real-time
    if (stats) {
      const newStats = { ...stats };
      // Update based on event data
      setStats(newStats);
    }
  };

  const handleExecutionUpdate = (event: any) => {
    // Update recent executions list
    const { execution_id, status } = event.data;
    setRecentExecutions(prev => 
      prev.map(exec => 
        exec.id === execution_id ? { ...exec, status } : exec
      )
    );
    
    // Update statistics
    if (stats && (status === 'success' || status === 'failed')) {
      const newStats = { ...stats };
      if (status === 'success') {
        newStats.executions.success_24h++;
      } else {
        newStats.executions.failed_24h++;
      }
      newStats.executions.success_rate = 
        (newStats.executions.success_24h / 
        (newStats.executions.success_24h + newStats.executions.failed_24h)) * 100;
      setStats(newStats);
    }
  };

  const handleTaskDispatched = (event: any) => {
    // Add new execution to recent list
    const newExecution: RecentExecution = {
      id: event.data.execution_id,
      task_name: event.data.task_name,
      agent_name: event.data.agent_name,
      status: 'running',
      started_at: new Date().toISOString(),
    };
    
    setRecentExecutions(prev => [newExecution, ...prev.slice(0, 9)]);
  };

  // Computed values
  const successRateColor = useMemo(() => {
    if (!stats) return 'text-gray-500';
    const rate = stats.executions.success_rate;
    if (rate >= 95) return 'text-green-500';
    if (rate >= 80) return 'text-yellow-500';
    return 'text-red-500';
  }, [stats]);

  const healthStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy':
        return <CheckCircle className="text-green-500" size={20} />;
      case 'degraded':
        return <AlertTriangle className="text-yellow-500" size={20} />;
      case 'down':
        return <XCircle className="text-red-500" size={20} />;
      default:
        return <Clock className="text-gray-500" size={20} />;
    }
  };

  const formatDuration = (seconds: number) => {
    if (seconds < 60) return `${Math.round(seconds)}s`;
    if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
    return `${Math.round(seconds / 3600)}h ${Math.round((seconds % 3600) / 60)}m`;
  };

  const formatBytes = (bytes: number) => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;
    
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    
    return `${size.toFixed(2)} ${units[unitIndex]}`;
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="neu-card p-8">
          <RefreshCw className="animate-spin text-primary mb-4" size={48} />
          <p className="text-gray-600">{t('dashboard.loading')}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="dashboard-container space-y-6 animate-fadeIn">
      {/* Header */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-3xl font-bold gradient-text">{t('dashboard.title')}</h1>
          <p className="text-gray-600 mt-1">
            {t('dashboard.welcome_back')} • {new Date().toLocaleDateString()}
          </p>
        </div>
        
        <div className="flex items-center space-x-4">
          {/* Time Range Selector */}
          <div className="neu-button-group">
            {['24h', '7d', '30d'].map(range => (
              <button
                key={range}
                onClick={() => setTimeRange(range)}
                className={classNames(
                  'neu-button-small',
                  timeRange === range && 'active'
                )}
              >
                {t(`dashboard.time_range.${range}`)}
              </button>
            ))}
          </div>
          
          <button 
            onClick={refreshDashboardData}
            disabled={refreshing}
            className="neu-button flex items-center space-x-2"
          >
            <RefreshCw className={classNames(
              'transition-transform',
              refreshing && 'animate-spin'
            )} size={16} />
            <span>{t('common.refresh')}</span>
          </button>
        </div>
      </div>

      {/* Key Metrics Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Agents Card */}
        <div className="neu-card p-6 hover:scale-105 transition-transform">
          <div className="flex justify-between items-start mb-4">
            <div>
              <p className="text-sm text-gray-500">{t('dashboard.agents.title')}</p>
              <p className="text-3xl font-bold">{stats?.agents.total || 0}</p>
            </div>
            <Server className="text-blue-500" size={32} />
          </div>
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-green-500">● {t('dashboard.agents.online')}</span>
              <span className="font-semibold">{stats?.agents.online || 0}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-blue-500">● {t('dashboard.agents.running')}</span>
              <span className="font-semibold">{stats?.agents.running_task || 0}</span>
            </div>
          </div>
          <div className="mt-4">
            <div className="text-xs text-gray-500 mb-1">{t('dashboard.agents.availability')}</div>
            <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
              <div 
                className="bg-green-500 h-2 rounded-full transition-all"
                style={{ 
                  width: `${stats ? (stats.agents.online / stats.agents.total * 100) : 0}%` 
                }}
              />
            </div>
          </div>
        </div>

        {/* Tasks Card */}
        <div className="neu-card p-6 hover:scale-105 transition-transform">
          <div className="flex justify-between items-start mb-4">
            <div>
              <p className="text-sm text-gray-500">{t('dashboard.tasks.title')}</p>
              <p className="text-3xl font-bold">{stats?.tasks.total || 0}</p>
            </div>
            <Calendar className="text-purple-500" size={32} />
          </div>
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span>{t('dashboard.tasks.active')}</span>
              <span className="font-semibold">{stats?.tasks.active || 0}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span>{t('dashboard.tasks.scheduled_today')}</span>
              <span className="font-semibold">{stats?.tasks.scheduled_today || 0}</span>
            </div>
          </div>
          <div className="mt-4 p-2 bg-purple-50 dark:bg-purple-900/20 rounded">
            <div className="text-xs text-purple-600 dark:text-purple-400">
              {t('dashboard.tasks.next_run')}: <span className="font-semibold">2 {t('common.minutes')}</span>
            </div>
          </div>
        </div>

        {/* Executions Card */}
        <div className="neu-card p-6 hover:scale-105 transition-transform">
          <div className="flex justify-between items-start mb-4">
            <div>
              <p className="text-sm text-gray-500">{t('dashboard.executions.title_24h')}</p>
              <p className="text-3xl font-bold">{stats?.executions.total_24h || 0}</p>
            </div>
            <Activity className="text-green-500" size={32} />
          </div>
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-green-500">✓ {t('dashboard.executions.success')}</span>
              <span className="font-semibold">{stats?.executions.success_24h || 0}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-red-500">✗ {t('dashboard.executions.failed')}</span>
              <span className="font-semibold">{stats?.executions.failed_24h || 0}</span>
            </div>
          </div>
          <div className="mt-4">
            <div className="flex justify-between items-center">
              <span className="text-xs text-gray-500">{t('dashboard.executions.success_rate')}</span>
              <span className={classNames('text-lg font-bold', successRateColor)}>
                {stats?.executions.success_rate.toFixed(1) || 0}%
              </span>
            </div>
          </div>
        </div>

        {/* Storage Card */}
        <div className="neu-card p-6 hover:scale-105 transition-transform">
          <div className="flex justify-between items-start mb-4">
            <div>
              <p className="text-sm text-gray-500">{t('dashboard.storage.title')}</p>
              <p className="text-3xl font-bold">
                {formatBytes(stats?.storage.total_backup_size || 0)}
              </p>
            </div>
            <Database className="text-orange-500" size={32} />
          </div>
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span>{t('dashboard.storage.daily_growth')}</span>
              <span className="font-semibold flex items-center">
                {stats?.storage.daily_growth > 0 ? (
                  <ArrowUp className="text-green-500" size={14} />
                ) : (
                  <ArrowDown className="text-red-500" size={14} />
                )}
                {formatBytes(Math.abs(stats?.storage.daily_growth || 0))}
              </span>
            </div>
            <div className="text-sm">
              <span className="text-gray-500">{t('dashboard.storage.most_used')}:</span>
              <span className="ml-2 font-semibold">{stats?.storage.most_used_remote || 'N/A'}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Execution Trend Chart */}
        <div className="neu-card p-6">
          <h3 className="text-lg font-semibold mb-4">{t('dashboard.charts.execution_trend')}</h3>
          <ResponsiveContainer width="100%" height={250}>
            <AreaChart data={chartData}>
              <defs>
                <linearGradient id="colorSuccess" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#10b981" stopOpacity={0.8}/>
                  <stop offset="95%" stopColor="#10b981" stopOpacity={0}/>
                </linearGradient>
                <linearGradient id="colorFailed" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#ef4444" stopOpacity={0.8}/>
                  <stop offset="95%" stopColor="#ef4444" stopOpacity={0}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
              <XAxis dataKey="time" stroke="#6b7280" />
              <YAxis stroke="#6b7280" />
              <Tooltip 
                contentStyle={{ 
                  backgroundColor: 'var(--neu-background)', 
                  border: 'none',
                  borderRadius: '8px',
                  boxShadow: 'var(--neu-shadow-small)'
                }}
              />
              <Legend />
              <Area 
                type="monotone" 
                dataKey="success" 
                stroke="#10b981" 
                fillOpacity={1} 
                fill="url(#colorSuccess)" 
                name={t('dashboard.charts.success')}
              />
              <Area 
                type="monotone" 
                dataKey="failed" 
                stroke="#ef4444" 
                fillOpacity={1} 
                fill="url(#colorFailed)" 
                name={t('dashboard.charts.failed')}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Task Distribution Chart */}
        <div className="neu-card p-6">
          <h3 className="text-lg font-semibold mb-4">{t('dashboard.charts.task_distribution')}</h3>
          <ResponsiveContainer width="100%" height={250}>
            <PieChart>
              <Pie
                data={[
                  { name: t('dashboard.charts.daily'), value: 40, color: '#3b82f6' },
                  { name: t('dashboard.charts.weekly'), value: 30, color: '#8b5cf6' },
                  { name: t('dashboard.charts.monthly'), value: 20, color: '#ec4899' },
                  { name: t('dashboard.charts.manual'), value: 10, color: '#f59e0b' },
                ]}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                outerRadius={80}
                fill="#8884d8"
                dataKey="value"
              >
                {[0, 1, 2, 3].map((index) => (
                  <Cell key={`cell-${index}`} fill={['#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b'][index]} />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Recent Executions & System Health Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Recent Executions */}
        <div className="lg:col-span-2 neu-card p-6">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-lg font-semibold">{t('dashboard.recent_executions.title')}</h3>
            <a href="/executions" className="text-primary hover:underline flex items-center">
              {t('dashboard.recent_executions.view_all')}
              <ChevronRight size={16} />
            </a>
          </div>
          
          <div className="space-y-3">
            {recentExecutions.map(execution => (
              <div 
                key={execution.id} 
                className="flex items-center justify-between p-3 rounded-lg bg-gray-50 dark:bg-gray-800 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
              >
                <div className="flex items-center space-x-3">
                  {execution.status === 'running' && <RefreshCw className="text-blue-500 animate-spin" size={18} />}
                  {execution.status === 'success' && <CheckCircle className="text-green-500" size={18} />}
                  {execution.status === 'failed' && <XCircle className="text-red-500" size={18} />}
                  {execution.status === 'pending' && <Clock className="text-yellow-500" size={18} />}
                  
                  <div>
                    <p className="font-medium">{execution.task_name}</p>
                    <p className="text-xs text-gray-500">
                      {execution.agent_name} • {new Date(execution.started_at).toLocaleTimeString()}
                    </p>
                  </div>
                </div>
                
                <div className="text-right">
                  {execution.duration && (
                    <span className="text-sm text-gray-600">
                      {formatDuration(execution.duration)}
                    </span>
                  )}
                  {execution.status === 'running' && (
                    <div className="flex items-center space-x-1">
                      <div className="w-2 h-2 bg-blue-500 rounded-full animate-pulse" />
                      <span className="text-xs text-blue-500">{t('common.running')}</span>
                    </div>
                  )}
                </div>
              </div>
            ))}
            
            {recentExecutions.length === 0 && (
              <div className="text-center py-8 text-gray-500">
                {t('dashboard.recent_executions.no_executions')}
              </div>
            )}
          </div>
        </div>

        {/* System Health */}
        <div className="neu-card p-6">
          <h3 className="text-lg font-semibold mb-4">{t('dashboard.system_health.title')}</h3>
          
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <span className="text-sm">{t('dashboard.system_health.hub')}</span>
              {healthStatusIcon(systemHealth?.hub_status || 'healthy')}
            </div>
            
            <div className="flex justify-between items-center">
              <span className="text-sm">{t('dashboard.system_health.database')}</span>
              {healthStatusIcon(systemHealth?.database_status || 'healthy')}
            </div>
            
            <div className="pt-4 border-t border-gray-200 dark:border-gray-700">
              <div className="space-y-3">
                <div>
                  <div className="flex justify-between text-sm mb-1">
                    <span className="text-gray-500">{t('dashboard.system_health.agent_response')}</span>
                    <span className="font-semibold">
                      {systemHealth?.avg_agent_response_time.toFixed(0) || 0}ms
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                    <div 
                      className={classNames(
                        'h-2 rounded-full transition-all',
                        systemHealth?.avg_agent_response_time < 100 
                          ? 'bg-green-500' 
                          : systemHealth?.avg_agent_response_time < 500
                          ? 'bg-yellow-500'
                          : 'bg-red-500'
                      )}
                      style={{ 
                        width: `${Math.min(100, (systemHealth?.avg_agent_response_time || 0) / 10)}%` 
                      }}
                    />
                  </div>
                </div>
                
                <div>
                  <div className="flex justify-between text-sm mb-1">
                    <span className="text-gray-500">{t('dashboard.system_health.system_load')}</span>
                    <span className="font-semibold">
                      {systemHealth?.system_load.toFixed(1) || 0}%
                    </span>
                  </div>
                  <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                    <div 
                      className={classNames(
                        'h-2 rounded-full transition-all',
                        systemHealth?.system_load < 50 
                          ? 'bg-green-500' 
                          : systemHealth?.system_load < 80
                          ? 'bg-yellow-500'
                          : 'bg-red-500'
                      )}
                      style={{ width: `${systemHealth?.system_load || 0}%` }}
                    />
                  </div>
                </div>
              </div>
            </div>

            <div className="pt-4 mt-4 border-t border-gray-200 dark:border-gray-700">
              <button className="neu-button-small w-full flex items-center justify-center space-x-2">
                <Zap size={14} />
                <span className="text-xs">{t('dashboard.system_health.run_diagnostics')}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="neu-card p-6">
        <h3 className="text-lg font-semibold mb-4">{t('dashboard.quick_actions.title')}</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <button className="neu-button flex flex-col items-center py-4 space-y-2">
            <Server size={24} />
            <span className="text-sm">{t('dashboard.quick_actions.add_agent')}</span>
          </button>
          <button className="neu-button flex flex-col items-center py-4 space-y-2">
            <Calendar size={24} />
            <span className="text-sm">{t('dashboard.quick_actions.create_task')}</span>
          </button>
          <button className="neu-button flex flex-col items-center py-4 space-y-2">
            <Database size={24} />
            <span className="text-sm">{t('dashboard.quick_actions.add_remote')}</span>
          </button>
          <button className="neu-button flex flex-col items-center py-4 space-y-2">
            <Activity size={24} />
            <span className="text-sm">{t('dashboard.quick_actions.view_logs')}</span>
          </button>
        </div>
      </div>
    </div>
  );
};

export default DashboardComplete;