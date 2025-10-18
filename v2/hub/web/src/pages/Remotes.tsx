import React, { useState, useEffect } from 'react';
import { Card, Table, Button, Modal, Form, Input, Space, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { apiService } from '../services/api';

interface Remote {
  id: string;
  name: string;
  type: string;
  created_at: string;
  updated_at: string;
}

const Remotes: React.FC = () => {
  const [remotes, setRemotes] = useState<Remote[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRemote, setEditingRemote] = useState<Remote | null>(null);
  const [form] = Form.useForm();

  useEffect(() => {
    fetchRemotes();
  }, []);

  const fetchRemotes = async () => {
    setLoading(true);
    try {
      const response = await apiService.getRemotes();
      setRemotes(response.data || []);
    } catch (error) {
      message.error('Failed to load remotes');
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (values: any) => {
    try {
      if (editingRemote) {
        await apiService.updateRemote(editingRemote.id, values);
        message.success('Remote updated successfully');
      } else {
        await apiService.createRemote(values);
        message.success('Remote created successfully');
      }
      setModalVisible(false);
      form.resetFields();
      setEditingRemote(null);
      fetchRemotes();
    } catch (error) {
      message.error('Operation failed');
    }
  };

  const handleEdit = (remote: Remote) => {
    setEditingRemote(remote);
    form.setFieldsValue(remote);
    setModalVisible(true);
  };

  const handleDelete = async (id: string) => {
    Modal.confirm({
      title: 'Confirm Delete',
      content: 'Are you sure you want to delete this remote?',
      onOk: async () => {
        try {
          await apiService.deleteRemote(id);
          message.success('Remote deleted successfully');
          fetchRemotes();
        } catch (error) {
          message.error('Failed to delete remote');
        }
      },
    });
  };

  const columns = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (text: string) => new Date(text).toLocaleDateString(),
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, record: Remote) => (
        <Space>
          <Button
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          />
          <Button
            danger
            icon={<DeleteOutlined />}
            onClick={() => handleDelete(record.id)}
          />
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card
        title="Rclone Remotes"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setEditingRemote(null);
              form.resetFields();
              setModalVisible(true);
            }}
          >
            Add Remote
          </Button>
        }
      >
        <Table
          dataSource={remotes}
          columns={columns}
          loading={loading}
          rowKey="id"
        />
      </Card>

      <Modal
        title={editingRemote ? 'Edit Remote' : 'Add Remote'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          form.resetFields();
          setEditingRemote(null);
        }}
        onOk={() => form.submit()}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Form.Item
            label="Name"
            name="name"
            rules={[{ required: true, message: 'Please enter remote name' }]}
          >
            <Input />
          </Form.Item>
          
          <Form.Item
            label="Type"
            name="type"
            rules={[{ required: true, message: 'Please enter remote type' }]}
          >
            <Input placeholder="e.g., s3, drive, dropbox" />
          </Form.Item>
          
          <Form.Item
            label="Configuration"
            name="config"
            rules={[{ required: true, message: 'Please enter configuration' }]}
          >
            <Input.TextArea rows={4} placeholder="Rclone configuration" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Remotes;