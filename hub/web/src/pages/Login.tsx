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
    <div className="d-flex flex-column justify-content-center align-items-center min-vh-100 py-4">
      <div className="text-center mb-4">
        <h2 className="navbar-brand navbar-brand-autodark fs-1 mb-2">
          {t('app.title')}
        </h2>
        <p className="text-muted fs-3">{t('auth.login.title')}</p>
      </div>

      <div className="card shadow-lg border-0" style={{ width: '100%', maxWidth: '420px' }}>
        <div className="card-body p-md-5">
          <form onSubmit={handleSubmit}>
            <div className="mb-3">
              <label className="form-label">{t('auth.login.username')}</label>
              <div className="input-group input-group-flat">
                <span className="input-group-text bg-transparent border-end-0">
                  <IconUser size={18} className="text-muted" />
                </span>
                <input
                  type="text"
                  className="form-control border-start-0 ps-1"
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
                <span className="input-group-text bg-transparent border-end-0">
                  <IconLock size={18} className="text-muted" />
                </span>
                <input
                  type="password"
                  className="form-control border-start-0 ps-1"
                  placeholder={t('login.password_placeholder')}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  autoComplete="current-password"
                />
              </div>
            </div>

            {error && (
              <div className="alert alert-danger" role="alert">
                {error}
              </div>
            )}

            <div className="form-footer mt-4">
              <button
                type="submit"
                className="btn btn-primary w-100 py-2 fs-3 fw-bold"
                disabled={loading}
              >
                {loading ? (
                  <div className="spinner-border spinner-border-sm me-2" role="status"></div>
                ) : null}
                {loading ? t('login.logging_in') : t('auth.login.submit')}
              </button>
            </div>
          </form>
        </div>
      </div>

      <div className="text-center text-muted mt-4 small">
        {t('login.default_hint')}
      </div>
    </div>
  );
};

export default Login;