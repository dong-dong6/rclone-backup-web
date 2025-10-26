import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Button, Badge, Modal, Input, Space, Typography, Row, Col, Statistic, Tag, Tooltip, message } from 'antd';
import { CloudServerOutlined, PlusOutlined, DeleteOutlined, EyeOutlined, CopyOutlined, CodeOutlined } from '@ant-design/icons';
import { apiService } from '../services/api';
import { useSSE } from '../contexts/SSEContext';

interface Agent {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'running_task';
  last_heartbeat: string | null;
  created_at: string;
  task_count?: number;
}

const { Title, Text, Paragraph } = Typography;

const Agents: React.FC = () => {
  const { t } = useTranslation();
  const { subscribe } = useSSE();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [showRegisterModal, setShowRegisterModal] = useState(false);
  const [registrationToken, setRegistrationToken] = useState('');
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null);

  useEffect(() => {
    loadAgents();
    
    // Subscribe to agent status updates
    const unsubscribe = subscribe('agent.status.update', (event) => {
      const { agent_id, status } = event.data;
      setAgents(prev => prev.map(agent => 
        agent.id === agent_id ? { ...agent, status } : agent
      ));
    });
    
    return () => {
      unsubscribe();
    };
  }, []);

  const loadAgents = async () => {
    try {
      setLoading(true);
      const response = await apiService.getAgents();
      setAgents(response.data);
    } catch (error) {
      console.error('Failed to load agents:', error);
    } finally {
      setLoading(false);
    }
  };

  const generateToken = async () => {
    try {
      const response = await apiService.createRegistrationToken();
      setRegistrationToken(response.data.token);
      setShowRegisterModal(true);
    } catch (error) {
      console.error('Failed to generate token:', error);
      message.error('生成注册令牌失败');
    }
  };

  const deleteAgent = async (id: string, name: string) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除节点 "${name}" 吗？此操作不可撤销。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await apiService.deleteAgent(id);
          await loadAgents();
          message.success('节点删除成功');
        } catch (error) {
          console.error('Failed to delete agent:', error);
          message.error('删除节点失败');
        }
      },
    });
  };

  const getStatusBadge = (status: string) => {
    const statusConfig = {
      online: { color: 'success', text: '在线' },
      offline: { color: 'error', text: '离线' },
      running_task: { color: 'processing', text: '运行中' },
    };
    
    const config = statusConfig[status as keyof typeof statusConfig] || statusConfig.offline;
    
    return (
      <Tag color={config.color}>
        {config.text}
      </Tag>
    );
  };

  const formatLastHeartbeat = (heartbeat: string | null) => {
    if (!heartbeat) return '从未';
    
    const date = new Date(heartbeat);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    
    if (diffMins < 1) return '刚刚';
    if (diffMins < 60) return `${diffMins}分钟前`;
    
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}小时前`;
    
    const diffDays = Math.floor(diffHours / 24);
    return `${diffDays}天前`;
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    message.success('已复制到剪贴板');
  };

  const generateRegistrationCommand = () => {
    const baseUrl = window.location.origin;
    const token = registrationToken;
    
    return `curl -sSL "${baseUrl}/api/v1/agent/install.sh" | sudo bash -s -- \\
  --hub-url "${baseUrl}" \\
  --token "${token}" \\
  --name "my-new-agent"`;
  };

  return (
    <div style={{ padding: '24px' }}>
      <div style={{ 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center', 
        marginBottom: '24px' 
      }}>
        <Title level={2} style={{ margin: 0 }}>
          <CloudServerOutlined style={{ marginRight: '8px' }} />
          节点管理
        </Title>
        <Button 
          type="primary" 
          icon={<PlusOutlined />} 
          onClick={generateToken}
          size="large"
        >
          注册新节点
        </Button>
      </div>

      {loading ? (
        <Card loading>
          <div style={{ height: '200px' }} />
        </Card>
      ) : agents.length === 0 ? (
        <Card>
          <div style={{ 
            textAlign: 'center', 
            padding: '48px 24px',
            color: '#8c8c8c'
          }}>
            <CloudServerOutlined style={{ fontSize: '48px', marginBottom: '16px' }} />
            <Title level={4} type="secondary">暂无注册节点</Title>
            <Paragraph type="secondary">
              点击上方按钮生成注册令牌，然后按照说明注册新节点
            </Paragraph>
            <Button type="primary" onClick={generateToken}>
              立即注册
            </Button>
          </div>
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {agents.map(agent => (
            <Col xs={24} sm={12} lg={8} xl={6} key={agent.id}>
              <Card
                hoverable
                actions={[
                  <Tooltip title="查看详情">
                    <EyeOutlined onClick={() => setSelectedAgent(agent)} />
                  </Tooltip>,
                  <Tooltip title="删除节点">
                    <DeleteOutlined 
                      onClick={() => deleteAgent(agent.id, agent.name)}
                      style={{ color: '#ff4d4f' }}
                    />
                  </Tooltip>
                ]}
              >
                <div style={{ textAlign: 'center' }}>
                  <div style={{ marginBottom: '16px' }}>
                    <CloudServerOutlined style={{ fontSize: '32px', color: '#1890ff' }} />
                  </div>
                  <Title level={4} style={{ margin: '0 0 8px 0' }}>
                    {agent.name}
                  </Title>
                  {getStatusBadge(agent.status)}
                </div>
                
                <div style={{ marginTop: '16px' }}>
                  <Row gutter={8}>
                    <Col span={12}>
                      <Statistic 
                        title="任务数量" 
                        value={agent.task_count || 0} 
                        valueStyle={{ fontSize: '16px' }}
                      />
                    </Col>
                    <Col span={12}>
                      <div>
                        <Text type="secondary" style={{ fontSize: '12px' }}>最后心跳</Text>
                        <div style={{ fontSize: '12px', marginTop: '2px' }}>
                          {formatLastHeartbeat(agent.last_heartbeat)}
                        </div>
                      </div>
                    </Col>
                  </Row>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      )}

      {/* Registration Modal */}
      <Modal
        title="注册新节点"
        open={showRegisterModal}
        onCancel={() => setShowRegisterModal(false)}
        width={800}
        footer={[
          <Button key="close" onClick={() => setShowRegisterModal(false)}>
            关闭
          </Button>
        ]}
      >
        <div style={{ marginBottom: '24px' }}>
          <Title level={4}>1. 注册令牌</Title>
          <Input.Group compact>
            <Input
              value={registrationToken}
              readOnly
              style={{ width: 'calc(100% - 100px)' }}
            />
            <Button 
              icon={<CopyOutlined />}
              onClick={() => copyToClipboard(registrationToken)}
            >
              复制令牌
            </Button>
          </Input.Group>
        </div>

        <div>
          <Title level={4}>2. 在目标机器上执行安装命令</Title>
          <Card>
            <pre style={{ 
              background: '#f5f5f5', 
              padding: '12px', 
              borderRadius: '4px',
              overflow: 'auto',
              whiteSpace: 'pre-wrap'
            }}>
              {generateRegistrationCommand()}
            </pre>
            <div style={{ marginTop: '12px' }}>
              <Button 
                type="primary" 
                icon={<CopyOutlined />}
                onClick={() => copyToClipboard(generateRegistrationCommand())}
              >
                复制命令
              </Button>
            </div>
          </Card>
        </div>

        <div style={{ marginTop: '24px', padding: '16px', background: '#f6ffed', borderRadius: '6px' }}>
          <Text type="secondary">
            <strong>说明：</strong>
            复制上面的命令到您想安装代理的机器上，并以 root 权限执行。脚本将自动下载、配置并启动代理服务。
          </Text>
        </div>
      </Modal>

      {/* Agent Details Modal */}
      <Modal
        title={`节点详情: ${selectedAgent?.name}`}
        open={!!selectedAgent}
        onCancel={() => setSelectedAgent(null)}
        width={800}
        footer={[
          <Button key="close" onClick={() => setSelectedAgent(null)}>
            关闭
          </Button>
        ]}
      >
        {selectedAgent && (
          <div>
            <Card title="基本信息" style={{ marginBottom: '16px' }}>
              <Row gutter={[16, 16]}>
                <Col span={12}>
                  <Text strong>节点ID:</Text>
                  <br />
                  <Text code>{selectedAgent.id}</Text>
                </Col>
                <Col span={12}>
                  <Text strong>状态:</Text>
                  <br />
                  {getStatusBadge(selectedAgent.status)}
                </Col>
                <Col span={12}>
                  <Text strong>最后心跳:</Text>
                  <br />
                  <Text>{formatLastHeartbeat(selectedAgent.last_heartbeat)}</Text>
                </Col>
                <Col span={12}>
                  <Text strong>创建时间:</Text>
                  <br />
                  <Text>{new Date(selectedAgent.created_at).toLocaleString()}</Text>
                </Col>
              </Row>
            </Card>
            
            <Card title="分配的任务" style={{ marginBottom: '16px' }}>
              <div style={{ textAlign: 'center', padding: '24px', color: '#8c8c8c' }}>
                <Text type="secondary">暂无分配的任务</Text>
              </div>
            </Card>
            
            <Card title="执行历史">
              <div style={{ textAlign: 'center', padding: '24px', color: '#8c8c8c' }}>
                <Text type="secondary">暂无执行记录</Text>
              </div>
            </Card>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default Agents;