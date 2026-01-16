import React from 'react';
import { useTranslation } from 'react-i18next';
import { IconAlertTriangle, IconRefresh } from '@tabler/icons-react';

export interface ErrorFallbackProps {
  error: Error;
  resetError: () => void;
}

export const ErrorFallback: React.FC<ErrorFallbackProps> = ({
  error,
  resetError,
}) => {
  const { t } = useTranslation();

  return (
    <div className="container-xl py-5">
      <div className="card">
        <div className="card-body text-center py-5">
          <IconAlertTriangle size={64} className="text-danger mb-3" />
          <h2 className="mb-3">{t('errors.something_went_wrong')}</h2>
          <p className="text-muted mb-4">
            {error.message || t('errors.unknown_error')}
          </p>
          <button className="btn btn-primary" onClick={resetError}>
            <IconRefresh size={16} className="me-2" />
            {t('common.retry')}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ErrorFallback;
