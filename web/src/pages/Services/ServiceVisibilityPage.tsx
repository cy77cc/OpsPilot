import React, { useEffect, useState } from 'react';
import { Button, Card, Form, Input, Select, Space, Typography, message } from 'antd';
import { useNavigate, useParams } from 'react-router-dom';
import { Api } from '../../api';
import type { ServiceVisibility } from '../../api/modules/services';

const { Title, Paragraph } = Typography;

const ServiceVisibilityPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [messageApi, contextHolder] = message.useMessage();
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!id) return;
    void Api.services.getDetail(id).then((resp) => {
      form.setFieldsValue({
        visibility: resp.data.visibility || 'team',
        granted_teams_text: (resp.data.grantedTeams || []).join(','),
      });
    });
  }, [id, form]);

  const save = async () => {
    if (!id) return;
    const values = await form.validateFields();
    setSaving(true);
    try {
      await Api.services.updateVisibility(id, values.visibility as ServiceVisibility);
      const teams = String(values.granted_teams_text || '')
        .split(',')
        .map((x) => x.trim())
        .filter(Boolean)
        .map((x) => Number(x))
        .filter((x) => Number.isFinite(x) && x > 0);
      await Api.services.updateGrantedTeams(id, teams);
      messageApi.success('可见性设置已更新');
      navigate(`/services/${id}`);
    } catch (err) {
      messageApi.error(err instanceof Error ? err.message : '保存失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="p-6 space-y-4">
      {contextHolder}
      <Title level={3}>服务可见性设置</Title>
      <Paragraph className="text-gray-500">支持 private、team、team-granted、public 四级可见性。</Paragraph>

      <Card>
        <Form form={form} layout="vertical">
          <Form.Item name="visibility" label="可见性" rules={[{ required: true }]}>
            <Select
              options={[
                { label: 'private', value: 'private' },
                { label: 'team', value: 'team' },
                { label: 'team-granted', value: 'team-granted' },
                { label: 'public', value: 'public' },
              ]}
            />
          </Form.Item>
          <Form.Item name="granted_teams_text" label="授权团队 ID 列表（逗号分隔）">
            <Input placeholder="例如: 2,3,5" />
          </Form.Item>
        </Form>

        <Space>
          <Button onClick={() => navigate(`/services/${id}`)}>取消</Button>
          <Button type="primary" loading={saving} onClick={save}>保存</Button>
        </Space>
      </Card>
    </div>
  );
};

export default ServiceVisibilityPage;
