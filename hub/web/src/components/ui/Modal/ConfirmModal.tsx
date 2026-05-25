import React from 'react';
import { Button, Space } from 'antd';
import { useTranslation } from 'react-i18next';
import { Modal } from './Modal';

export interface ConfirmModalProps {
  isOpen: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  confirmVariant?: 'primary' | 'danger';
  loading?: boolean;
}

export const ConfirmModal: React.FC<ConfirmModalProps> = ({
  isOpen,
  onConfirm,
  onCancel,
  title,
  message,
  confirmText,
  cancelText,
  confirmVariant = 'primary',
  loading = false,
}) => {
  const { t } = useTranslation();

  const footer = (
    <Space>
      <Button onClick={onCancel} disabled={loading}>
        {cancelText || t('common.cancel')}
      </Button>
      <Button
        type="primary"
        danger={confirmVariant === 'danger'}
        onClick={onConfirm}
        loading={loading}
      >
        {confirmText || t('common.confirm')}
      </Button>
    </Space>
  );

  return (
    <Modal
      isOpen={isOpen}
      onClose={onCancel}
      title={title}
      size="sm"
      footer={footer}
      loading={loading}
    >
      <p>{message}</p>
    </Modal>
  );
};

export default ConfirmModal;
