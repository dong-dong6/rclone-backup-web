import React from 'react';
import { Flex, Spin, Typography } from 'antd';

export type LoadingSize = 'sm' | 'md' | 'lg';

export interface LoadingProps {
  size?: LoadingSize;
  text?: string;
  fullPage?: boolean;
}

const sizeMap: Record<LoadingSize, 'small' | 'default' | 'large'> = {
  sm: 'small',
  md: 'default',
  lg: 'large',
};

export const Loading: React.FC<LoadingProps> = ({
  size = 'md',
  text,
  fullPage = false,
}) => {
  const content = (
    <Flex vertical align="center" justify="center" gap={12} style={{ padding: 48 }}>
      <Spin size={sizeMap[size]} />
      {text && <Typography.Text type="secondary">{text}</Typography.Text>}
    </Flex>
  );

  if (fullPage) {
    return (
      <Flex align="center" justify="center" style={{ minHeight: '100vh' }}>
        {content}
      </Flex>
    );
  }

  return content;
};

export default Loading;
