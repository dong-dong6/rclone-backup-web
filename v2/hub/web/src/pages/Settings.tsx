import React from 'react';
import { Card, Form, Input, Button, Switch, Tabs, message } from 'antd';

const Settings: React.FC = () => {
  const [form] = Form.useForm();

  const handleSave = async (values: any) => {
    try {
      // TODO: Implement settings save
      console.log('Saving settings:', values);
      message.success('Settings saved successfully');
    } catch (error) {
      message.error('Failed to save settings');
    }
  };

  const generalSettings = (
    <Form
      form={form}
      layout="vertical"
      initialValues={{
        hub_name: 'Rclone Backup Hub',
        session_timeout: 24,
        log_level: 'info',
        enable_metrics: true,
      }}
      onFinish={handleSave}
    >
      <Form.Item
        label="Hub Name"
        name="hub_name"
      >
        <Input />
      </Form.Item>
      
      <Form.Item
        label="Session Timeout (hours)"
        name="session_timeout"
      >
        <Input type="number" />
      </Form.Item>
      
      <Form.Item
        label="Log Level"
        name="log_level"
      >
        <Input />
      </Form.Item>
      
      <Form.Item
        label="Enable Metrics"
        name="enable_metrics"
        valuePropName="checked"
      >
        <Switch />
      </Form.Item>
      
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Save Settings
        </Button>
      </Form.Item>
    </Form>
  );

  const securitySettings = (
    <Form
      layout="vertical"
      onFinish={handleSave}
    >
      <Form.Item
        label="Current Password"
        name="current_password"
        rules={[{ required: true, message: 'Please enter current password' }]}
      >
        <Input.Password />
      </Form.Item>
      
      <Form.Item
        label="New Password"
        name="new_password"
        rules={[{ required: true, message: 'Please enter new password' }]}
      >
        <Input.Password />
      </Form.Item>
      
      <Form.Item
        label="Confirm Password"
        name="confirm_password"
        rules={[{ required: true, message: 'Please confirm password' }]}
      >
        <Input.Password />
      </Form.Item>
      
      <Form.Item>
        <Button type="primary" htmlType="submit">
          Change Password
        </Button>
      </Form.Item>
    </Form>
  );

  const items = [
    {
      key: 'general',
      label: 'General',
      children: generalSettings,
    },
    {
      key: 'security',
      label: 'Security',
      children: securitySettings,
    },
  ];

  return (
    <Card title="Settings">
      <Tabs items={items} />
    </Card>
  );
};

export default Settings;