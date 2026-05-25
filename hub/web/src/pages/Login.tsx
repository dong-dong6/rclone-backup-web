import React from 'react';
import { Alert, Button, Card, Form, Input, Typography } from 'antd';
import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../contexts/AuthContext';
import './Login.css';

interface LoginFormValues {
  username: string;
  password: string;
}

const Login: React.FC = () => {
  const navigate = useNavigate();
  const { login } = useAuth();
  const { t } = useTranslation();
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState('');

  const handleSubmit = async ({ username, password }: LoginFormValues) => {
    setLoading(true);
    setError('');

    try {
      await login(username, password);
      navigate('/');
    } catch {
      setError(t('auth.login.error'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-page">
      <div className="login-header">
        <Typography.Title level={2}>{t('app.title')}</Typography.Title>
        <Typography.Text type="secondary">{t('auth.login.title')}</Typography.Text>
      </div>

      <Card className="login-card">
        <Form layout="vertical" onFinish={handleSubmit} requiredMark={false}>
          <Form.Item
            label={t('auth.login.username')}
            name="username"
            rules={[{ required: true }]}
          >
            <Input
              prefix={<UserOutlined />}
              placeholder={t('login.username_placeholder')}
              autoComplete="username"
              size="large"
            />
          </Form.Item>

          <Form.Item
            label={t('auth.login.password')}
            name="password"
            rules={[{ required: true }]}
          >
            <Input.Password
              prefix={<LockOutlined />}
              placeholder={t('login.password_placeholder')}
              autoComplete="current-password"
              size="large"
            />
          </Form.Item>

          {error && <Alert type="error" message={error} showIcon style={{ marginBottom: 16 }} />}

          <Button type="primary" htmlType="submit" size="large" block loading={loading}>
            {loading ? t('login.logging_in') : t('auth.login.submit')}
          </Button>
        </Form>
      </Card>

      <Typography.Text type="secondary" className="login-hint">
        {t('login.default_hint')}
      </Typography.Text>
    </div>
  );
};

export default Login;
