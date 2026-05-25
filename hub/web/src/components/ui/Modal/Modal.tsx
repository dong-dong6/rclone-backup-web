import React from 'react';
import { Modal as AntModal } from 'antd';

export type ModalSize = 'sm' | 'md' | 'lg' | 'xl';

export interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  size?: ModalSize;
  children: React.ReactNode;
  footer?: React.ReactNode;
  loading?: boolean;
  closeOnOverlayClick?: boolean;
}

const widthMap: Record<ModalSize, number> = {
  sm: 420,
  md: 560,
  lg: 760,
  xl: 960,
};

export const Modal: React.FC<ModalProps> = ({
  isOpen,
  onClose,
  title,
  size = 'md',
  children,
  footer,
  loading = false,
  closeOnOverlayClick = true,
}) => {
  return (
    <AntModal
      open={isOpen}
      onCancel={onClose}
      title={title}
      width={widthMap[size]}
      footer={footer}
      confirmLoading={loading}
      maskClosable={closeOnOverlayClick && !loading}
      closable={!loading}
      destroyOnHidden
    >
      {children}
    </AntModal>
  );
};

export default Modal;
