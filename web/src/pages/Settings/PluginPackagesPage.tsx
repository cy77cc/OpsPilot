import React, { useCallback, useEffect, useState } from 'react';
import { Button, Card, Form, Input, Modal, Select, Table, Tag, Upload, message, Popconfirm } from 'antd';
import { UploadOutlined, DeleteOutlined } from '@ant-design/icons';
import { Api } from '../../api';
import type { HostPluginPackage } from '../../types/host';

const PluginPackagesPage: React.FC = () => {
  const [packages, setPackages] = useState<HostPluginPackage[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [form] = Form.useForm();

  const loadPackages = useCallback(async () => {
    setLoading(true);
    try {
      const res = await Api.hosts.listPluginPackages();
      setPackages(res.data || []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadPackages();
  }, [loadPackages]);

  const handleUpload = async () => {
    const values = await form.validateFields();
    if (!values.file?.fileList?.[0]) {
      message.error('请选择文件');
      return;
    }

    const formData = new FormData();
    formData.append('plugin_key', values.pluginKey);
    formData.append('version', values.version);
    formData.append('arch', values.arch);
    formData.append('file', values.file.fileList[0].originFileObj);

    setUploading(true);
    try {
      await Api.hosts.uploadPluginPackage(formData);
      message.success('上传成功');
      setUploadModalOpen(false);
      form.resetFields();
      void loadPackages();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '上传失败');
    } finally {
      setUploading(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await Api.hosts.deletePluginPackage(id);
      message.success('删除成功');
      void loadPackages();
    } catch (err) {
      message.error(err instanceof Error ? err.message : '删除失败');
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
  };

  return (
    <div>
      <Card
        title="插件包管理"
        extra={
          <Button type="primary" onClick={() => setUploadModalOpen(true)}>
            上传安装包
          </Button>
        }
      >
        <Table
          rowKey="id"
          loading={loading}
          dataSource={packages}
          pagination={false}
          columns={[
            { title: '插件', dataIndex: 'pluginKey' },
            { title: '版本', dataIndex: 'version' },
            { title: '架构', dataIndex: 'arch', render: (v: string) => <Tag>{v}</Tag> },
            { title: '文件名', dataIndex: 'filename' },
            { title: '大小', dataIndex: 'sizeBytes', render: (v: number) => formatBytes(v) },
            { title: '校验和', dataIndex: 'checksum', render: (v: string) => v.substring(0, 12) + '...' },
            {
              title: '操作',
              key: 'actions',
              render: (_: unknown, record: HostPluginPackage) => (
                <Popconfirm
                  title="确定删除此安装包？"
                  onConfirm={() => handleDelete(record.id)}
                  okText="删除"
                  cancelText="取消"
                  okButtonProps={{ danger: true }}
                >
                  <Button type="link" danger icon={<DeleteOutlined />} size="small">
                    删除
                  </Button>
                </Popconfirm>
              ),
            },
          ]}
          locale={{ emptyText: '暂无安装包' }}
        />
      </Card>

      <Modal
        title="上传安装包"
        open={uploadModalOpen}
        onCancel={() => setUploadModalOpen(false)}
        onOk={handleUpload}
        confirmLoading={uploading}
        okText="上传"
      >
        <Form form={form} layout="vertical" initialValues={{ pluginKey: 'opsagent', arch: 'amd64' }}>
          <Form.Item name="pluginKey" label="插件" rules={[{ required: true }]}>
            <Select options={[{ label: 'OpsAgent', value: 'opsagent' }]} />
          </Form.Item>
          <Form.Item name="version" label="版本" rules={[{ required: true, message: '请输入版本号' }]}>
            <Input placeholder="例如: v1.0.0" />
          </Form.Item>
          <Form.Item name="arch" label="架构" rules={[{ required: true }]}>
            <Select options={[{ label: 'amd64', value: 'amd64' }, { label: 'arm64', value: 'arm64' }]} />
          </Form.Item>
          <Form.Item name="file" label="安装包文件" rules={[{ required: true, message: '请选择文件' }]}>
            <Upload accept=".tar.gz,.tgz" maxCount={1} beforeUpload={() => false}>
              <Button icon={<UploadOutlined />}>选择文件</Button>
            </Upload>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default PluginPackagesPage;
