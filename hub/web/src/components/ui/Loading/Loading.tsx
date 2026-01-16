import React from 'react';
import { IconRefresh } from '@tabler/icons-react';

export type LoadingSize = 'sm' | 'md' | 'lg';

export interface LoadingProps {
  size?: LoadingSize;
  text?: string;
  fullPage?: boolean;
}

const sizeMap: Record<LoadingSize, number> = {
  sm: 24,
  md: 48,
  lg: 64,
};

export const Loading: React.FC<LoadingProps> = ({
  size = 'md',
  text,
  fullPage = false,
}) => {
  const content = (
    <div className="text-center py-5">
      <IconRefresh className="spinner text-primary mb-3" size={sizeMap[size]} />
      {text && <p className="text-muted">{text}</p>}
    </div>
  );

  if (fullPage) {
    return (
      <div className="d-flex align-items-center justify-content-center min-vh-100">
        {content}
      </div>
    );
  }

  return content;
};

export default Loading;
