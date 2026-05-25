import React from 'react';
import { Form, Radio } from 'antd';
import { DesktopOutlined, MoonOutlined, SunOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useTheme, type Theme } from '../../../contexts/ThemeContext';

export const ThemeSelector: React.FC = () => {
  const { t } = useTranslation();
  const { theme, setTheme } = useTheme();

  return (
    <Form.Item label={t('settings.theme') || 'Theme'}>
      <Radio.Group
        value={theme}
        onChange={(event) => setTheme(event.target.value as Theme)}
        optionType="button"
        buttonStyle="solid"
      >
        <Radio.Button value="light">
          <SunOutlined /> Light
        </Radio.Button>
        <Radio.Button value="dark">
          <MoonOutlined /> Dark
        </Radio.Button>
        <Radio.Button value="auto">
          <DesktopOutlined /> Auto
        </Radio.Button>
      </Radio.Group>
    </Form.Item>
  );
};
