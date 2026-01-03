import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../contexts/AuthContext';
import { useTheme } from '../contexts/ThemeContext';
import '../styles/neumorphism-bw.css';

const LoginBW: React.FC = () => {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { login } = useAuth();
  const { isDarkMode, toggleTheme } = useTheme();
  const [loading, setLoading] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [rememberMe, setRememberMe] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    
    try {
      await login(username, password);
      if (rememberMe) {
        localStorage.setItem('rememberUsername', username);
      }
      navigate('/');
    } catch (err) {
      setError(t('login.error') || 'Invalid username or password');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const savedUsername = localStorage.getItem('rememberUsername');
    if (savedUsername) {
      setUsername(savedUsername);
      setRememberMe(true);
    }
  }, []);

  return (
    <div className={`neu-login-container ${isDarkMode ? 'dark-mode' : ''}`} style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'var(--neu-bg-primary)',
      position: 'relative',
    }}>
      {/* Theme Toggle Button */}
      <button
        onClick={toggleTheme}
        className="neu-button"
        style={{
          position: 'absolute',
          top: '20px',
          right: '20px',
          width: '50px',
          height: '50px',
          borderRadius: '50%',
          padding: 0,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: '20px',
        }}
        aria-label="Toggle theme"
      >
        {isDarkMode ? '☀️' : '🌙'}
      </button>

      {/* Login Card */}
      <div className="neu-card neu-fade-in" style={{
        width: '100%',
        maxWidth: '420px',
        padding: '48px 40px',
      }}>
        {/* Logo and Title */}
        <div style={{ textAlign: 'center', marginBottom: '40px' }}>
          <div className="neu-avatar" style={{
            width: '80px',
            height: '80px',
            margin: '0 auto 24px',
            fontSize: '36px',
          }}>
            💾
          </div>
          <h1 style={{ 
            margin: '0 0 8px 0',
            fontSize: '28px',
            fontWeight: '700',
            color: 'var(--neu-text-primary)',
          }}>
            {t('app.title')}
          </h1>
          <p style={{ 
            margin: 0,
            fontSize: '14px',
            color: 'var(--neu-text-secondary)',
          }}>
            {t('login.subtitle') || '分布式备份管理系统'}
          </p>
        </div>

        {/* Login Form */}
        <form onSubmit={handleSubmit} style={{ marginTop: '32px' }}>
          {/* Username Input */}
          <div style={{ marginBottom: '24px' }}>
            <label style={{ 
              display: 'block',
              marginBottom: '10px',
              fontSize: '14px',
              fontWeight: '600',
              color: 'var(--neu-text-secondary)',
            }}>
              {t('login.username') || '用户名'}
            </label>
            <div style={{ position: 'relative' }}>
              <input
                type="text"
                className="neu-input"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder={t('login.username_placeholder') || '请输入用户名'}
                required
                autoComplete="username"
                style={{ paddingLeft: '44px' }}
              />
              <span style={{
                position: 'absolute',
                left: '16px',
                top: '50%',
                transform: 'translateY(-50%)',
                fontSize: '18px',
              }}>
                👤
              </span>
            </div>
          </div>

          {/* Password Input */}
          <div style={{ marginBottom: '24px' }}>
            <label style={{ 
              display: 'block',
              marginBottom: '10px',
              fontSize: '14px',
              fontWeight: '600',
              color: 'var(--neu-text-secondary)',
            }}>
              {t('login.password') || '密码'}
            </label>
            <div style={{ position: 'relative' }}>
              <input
                type="password"
                className="neu-input"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder={t('login.password_placeholder') || '请输入密码'}
                required
                autoComplete="current-password"
                style={{ paddingLeft: '44px' }}
              />
              <span style={{
                position: 'absolute',
                left: '16px',
                top: '50%',
                transform: 'translateY(-50%)',
                fontSize: '18px',
              }}>
                🔒
              </span>
            </div>
          </div>

          {/* Remember Me */}
          <div style={{ 
            marginBottom: '24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}>
            <label style={{
              display: 'flex',
              alignItems: 'center',
              cursor: 'pointer',
              fontSize: '14px',
              color: 'var(--neu-text-secondary)',
            }}>
              <div 
                className={`neu-checkbox ${rememberMe ? 'checked' : ''}`}
                onClick={() => setRememberMe(!rememberMe)}
                style={{ marginRight: '8px' }}
              />
              {t('login.remember_me') || '记住我'}
            </label>
            <a 
              href="#"
              style={{
                fontSize: '14px',
                color: 'var(--neu-text-secondary)',
                textDecoration: 'none',
              }}
              onClick={(e) => {
                e.preventDefault();
                alert(t('login.forgot_password_hint') || '请联系管理员重置密码');
              }}
            >
              {t('login.forgot_password') || '忘记密码？'}
            </a>
          </div>

          {/* Error Message */}
          {error && (
            <div className="neu-card" style={{
              marginBottom: '24px',
              padding: '12px 16px',
              borderRadius: '8px',
              background: 'var(--neu-bg-secondary)',
              boxShadow: 'var(--neu-shadow-concave)',
            }}>
              <span style={{
                color: 'var(--neu-text-primary)',
                fontSize: '14px',
                display: 'flex',
                alignItems: 'center',
              }}>
                ⚠️ <span style={{ marginLeft: '8px' }}>{error}</span>
              </span>
            </div>
          )}

          {/* Submit Button */}
          <button
            type="submit"
            className="neu-button neu-button-primary"
            disabled={loading}
            style={{
              width: '100%',
              padding: '14px',
              fontSize: '16px',
              fontWeight: '600',
              borderRadius: '12px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            {loading ? (
              <>
                <div className="neu-spinner" style={{
                  width: '20px',
                  height: '20px',
                  marginRight: '8px',
                }} />
                {t('login.logging_in') || '登录中...'}
              </>
            ) : (
              <>
                {t('login.submit') || '登录'}
                <span style={{ marginLeft: '8px' }}>→</span>
              </>
            )}
          </button>
        </form>

        {/* Divider */}
        <div className="neu-divider" style={{ margin: '32px 0' }} />

        {/* Footer */}
        <div style={{
          textAlign: 'center',
          fontSize: '13px',
          color: 'var(--neu-text-disabled)',
        }}>
          <p style={{ margin: '0 0 8px 0' }}>
            {t('login.default_hint') || '默认账号: admin / admin'}
          </p>
          <p style={{ margin: 0 }}>
            © 2025 Rclone Backup System
          </p>
        </div>
      </div>

      {/* Background Decoration */}
      <div style={{
        position: 'absolute',
        top: '10%',
        left: '10%',
        width: '100px',
        height: '100px',
        borderRadius: '50%',
        background: 'var(--neu-bg-primary)',
        boxShadow: 'var(--neu-shadow-subtle)',
        opacity: 0.5,
      }} />
      <div style={{
        position: 'absolute',
        bottom: '10%',
        right: '10%',
        width: '150px',
        height: '150px',
        borderRadius: '50%',
        background: 'var(--neu-bg-primary)',
        boxShadow: 'var(--neu-shadow-subtle)',
        opacity: 0.5,
      }} />
    </div>
  );
};

export default LoginBW;