import React from 'react';
import { IconUser, IconLock } from '@tabler/icons-react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../contexts/AuthContext';
import './Login.css';

const Login: React.FC = () => {
  const navigate = useNavigate();
  const { login } = useAuth();
  const { t } = useTranslation();
  const [loading, setLoading] = React.useState(false);
  const [username, setUsername] = React.useState('');
  const [password, setPassword] = React.useState('');
  const [error, setError] = React.useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    
    try {
      await login(username, password);
      navigate('/');
    } catch (error) {
      setError(t('auth.login.error'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="page page-center">
      <div className="container-tight py-4">
        <div className="text-center mb-4">
          <h2 className="h2 mb-2">{t('app.title')}</h2>
          <p className="text-muted">{t('auth.login.title')}</p>
        </div>
        
        <div className="card card-md">
          <div className="card-body">
            <form onSubmit={handleSubmit}>
              <div className="mb-3">
                <label className="form-label">{t('auth.login.username')}</label>
                <div className="input-group input-group-flat">
                  <span className="input-group-text">
                    <IconUser size={16} />
                  </span>
                  <input
                    type="text"
                    className="form-control"
                    placeholder={t('login.username_placeholder')}
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    required
                    autoComplete="username"
                  />
                </div>
              </div>

              <div className="mb-3">
                <label className="form-label">{t('auth.login.password')}</label>
                <div className="input-group input-group-flat">
                  <span className="input-group-text">
                    <IconLock size={16} />
                  </span>
                  <input
                    type="password"
                    className="form-control"
                    placeholder={t('login.password_placeholder')}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    autoComplete="current-password"
                  />
                </div>
              </div>

              {error && (
                <div className="alert alert-danger">
                  {error}
                </div>
              )}

              <div className="form-footer">
                <button
                  type="submit"
                  className="btn btn-primary w-100"
                  disabled={loading}
                >
                  {loading ? t('login.logging_in') : t('auth.login.submit')}
                </button>
              </div>
            </form>
          </div>
        </div>
        
        <div className="text-center text-muted mt-3">
          {t('login.default_hint')}
        </div>
      </div>
    </div>
  );
};

export default Login;