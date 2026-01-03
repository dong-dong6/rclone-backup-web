import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../contexts/AuthContext';
import '../styles/neumorphism.css';

const LoginNeu: React.FC = () => {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { login } = useAuth();
  const [loading, setLoading] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    
    try {
      await login(username, password);
      navigate('/');
    } catch (err) {
      setError(t('login.error') || 'Invalid username or password');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container" style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'linear-gradient(145deg, #e6e6e6, #f0f0f0)',
    }}>
      <div className="neu-card" style={{
        width: '400px',
        padding: '40px',
        borderRadius: '20px',
      }}>
        <div style={{ textAlign: 'center', marginBottom: '30px' }}>
          <h1 style={{ 
            margin: '0 0 10px 0',
            color: '#333',
            fontSize: '28px',
            fontWeight: '600',
          }}>
            {t('app.title')}
          </h1>
          <p style={{ 
            margin: 0,
            color: '#666',
            fontSize: '14px',
          }}>
            {t('login.subtitle') || '分布式备份管理系统'}
          </p>
        </div>

        <form onSubmit={handleSubmit}>
          <div style={{ marginBottom: '20px' }}>
            <label style={{ 
              display: 'block',
              marginBottom: '8px',
              color: '#555',
              fontSize: '14px',
              fontWeight: '500',
            }}>
              {t('login.username') || '用户名'}
            </label>
            <input
              type="text"
              className="neu-input"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={t('login.username_placeholder') || '请输入用户名'}
              required
              style={{
                width: '100%',
                padding: '12px 16px',
                fontSize: '14px',
                border: 'none',
                borderRadius: '10px',
                background: '#e0e0e0',
                boxShadow: 'inset 5px 5px 10px #bebebe, inset -5px -5px 10px #ffffff',
              }}
            />
          </div>

          <div style={{ marginBottom: '24px' }}>
            <label style={{ 
              display: 'block',
              marginBottom: '8px',
              color: '#555',
              fontSize: '14px',
              fontWeight: '500',
            }}>
              {t('login.password') || '密码'}
            </label>
            <input
              type="password"
              className="neu-input"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t('login.password_placeholder') || '请输入密码'}
              required
              style={{
                width: '100%',
                padding: '12px 16px',
                fontSize: '14px',
                border: 'none',
                borderRadius: '10px',
                background: '#e0e0e0',
                boxShadow: 'inset 5px 5px 10px #bebebe, inset -5px -5px 10px #ffffff',
              }}
            />
          </div>

          {error && (
            <div style={{
              marginBottom: '20px',
              padding: '10px',
              borderRadius: '8px',
              background: '#ffebee',
              color: '#c62828',
              fontSize: '14px',
              textAlign: 'center',
            }}>
              {error}
            </div>
          )}

          <button
            type="submit"
            className="neu-button neu-button-primary"
            disabled={loading}
            style={{
              width: '100%',
              padding: '12px',
              fontSize: '16px',
              fontWeight: '500',
              border: 'none',
              borderRadius: '10px',
              background: 'linear-gradient(145deg, #4a9eff, #1976d2)',
              color: 'white',
              cursor: loading ? 'not-allowed' : 'pointer',
              opacity: loading ? 0.7 : 1,
              boxShadow: loading ? 'none' : '5px 5px 10px #bebebe, -5px -5px 10px #ffffff',
              transition: 'all 0.3s ease',
            }}
          >
            {loading ? (t('login.logging_in') || '登录中...') : (t('login.submit') || '登录')}
          </button>
        </form>

        <div style={{
          marginTop: '30px',
          paddingTop: '20px',
          borderTop: '1px solid #ddd',
          textAlign: 'center',
          color: '#999',
          fontSize: '12px',
        }}>
          {t('login.default_hint') || '默认账号: admin / admin'}
        </div>
      </div>
    </div>
  );
};

export default LoginNeu;