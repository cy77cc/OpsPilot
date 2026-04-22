import React, { useEffect, useState } from 'react';
import { Button, Card, Col, Form, Input, Modal, Row, Space, Table, message } from 'antd';
import { Api } from '../../api';
import { GuidedFormItem } from '../../components/FormGuidance';

const AutomationPage: React.FC = () => {
  const [inventories, setInventories] = useState<any[]>([]);
  const [playbooks, setPlaybooks] = useState<any[]>([]);
  const [runLogs, setRunLogs] = useState<any[]>([]);
  const [invOpen, setInvOpen] = useState(false);
  const [pbOpen, setPbOpen] = useState(false);
  const [previewToken, setPreviewToken] = useState('');
  const [action, setAction] = useState('shell');
  const [hostIDs, setHostIDs] = useState('1');
  const [command, setCommand] = useState('hostname');
  const [invForm] = Form.useForm();
  const [pbForm] = Form.useForm();

  const load = async () => {
    try {
      const [invRes, pbRes] = await Promise.all([Api.automation.listInventories(), Api.automation.listPlaybooks()]);
      // 确保数据是数组，防止 Table 组件报错 (rawData.some is not a function)
      setInventories(Array.isArray(invRes.data) ? invRes.data : []);
      setPlaybooks(Array.isArray(pbRes.data) ? pbRes.data : []);
    } catch (err) {
      console.error('Failed to load automation data:', err);
      message.error('加载自动化数据失败');
      setInventories([]);
      setPlaybooks([]);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const createInventory = async () => {
    const values = await invForm.validateFields();
    await Api.automation.createInventory({ name: values.name, hostsJson: values.hostsJson || '[]' });
    message.success('Inventory 已创建');
    setInvOpen(false);
    invForm.resetFields();
    load();
  };

  const createPlaybook = async () => {
    const values = await pbForm.validateFields();
    await Api.automation.createPlaybook({ name: values.name, contentYml: values.contentYml });
    message.success('Playbook 已创建');
    setPbOpen(false);
    pbForm.resetFields();
    load();
  };

  const previewRun = async () => {
    const ids = hostIDs.split(',').map((x) => Number(x.trim())).filter((x) => x > 0);
    const res = await Api.automation.previewRun({ action, params: { target_count: ids.length || 1, host_ids: ids, command, content: command } });
    setPreviewToken(res.data.approval_token);
    message.success('已生成预执行 Token');
  };

  const executeRun = async () => {
    if (!previewToken) {
      message.warning('请先预执行');
      return;
    }
    const res = await Api.automation.executeRun({ approval_token: previewToken });
    const runId = String(res.data.runId || '');
    if (runId) {
      const logs = await Api.automation.getRunLogs(runId);
      setRunLogs(Array.isArray(logs.data) ? logs.data : []);
    }
    setPreviewToken('');
    message.success('执行成功');
  };

  return (
    <Row gutter={[16, 16]}>
      <Col span={12}>
        <Card title="Inventories" extra={<Button onClick={() => setInvOpen(true)}>新建</Button>}>
          <Table 
            rowKey="id" 
            pagination={false} 
            dataSource={inventories} 
            columns={[
              { title: 'ID', dataIndex: 'id' }, 
              { title: 'Name', dataIndex: 'name' }, 
              { title: 'Hosts', dataIndex: 'hosts_json' }
            ]} 
          />
        </Card>
      </Col>
      <Col span={12}>
        <Card title="Playbooks" extra={<Button onClick={() => setPbOpen(true)}>新建</Button>}>
          <Table 
            rowKey="id" 
            pagination={false} 
            dataSource={playbooks} 
            columns={[
              { title: 'ID', dataIndex: 'id' }, 
              { title: 'Name', dataIndex: 'name' }, 
              { title: 'Risk', dataIndex: 'risk_level' }
            ]} 
          />
        </Card>
      </Col>
      <Col span={24}>
        <Card title="执行控制台" extra={<Space><Button onClick={previewRun}>Preview</Button><Button type="primary" onClick={executeRun}>Execute</Button></Space>}>
          <Space direction="vertical" style={{ width: '100%' }}>
            <Input value={action} onChange={(e) => setAction(e.target.value)} placeholder="action: shell/script/playbook" />
            <Input value={hostIDs} onChange={(e) => setHostIDs(e.target.value)} placeholder="host ids: 1,2,3" />
            <Input.TextArea value={command} onChange={(e) => setCommand(e.target.value)} rows={3} placeholder="command or playbook content" />
          </Space>
          <p style={{ marginTop: 16 }}>approval_token: {previewToken || '-'}</p>
          <Table 
            rowKey="id" 
            pagination={false} 
            dataSource={runLogs} 
            columns={[
              { title: '时间', dataIndex: 'created_at' }, 
              { title: '级别', dataIndex: 'level' }, 
              { title: '内容', dataIndex: 'message' }
            ]} 
          />
        </Card>
      </Col>

      <Modal title="新建 Inventory" open={invOpen} onCancel={() => setInvOpen(false)} onOk={createInventory}>
        <Form form={invForm} layout="vertical">
          <GuidedFormItem name="name" label="名称" rules={[{ required: true }]}><Input /></GuidedFormItem>
          <GuidedFormItem name="hostsJson" label="Hosts(JSON)"><Input.TextArea rows={4} /></GuidedFormItem>
        </Form>
      </Modal>

      <Modal title="新建 Playbook" open={pbOpen} onCancel={() => setPbOpen(false)} onOk={createPlaybook}>
        <Form form={pbForm} layout="vertical">
          <GuidedFormItem name="name" label="名称" rules={[{ required: true }]}><Input /></GuidedFormItem>
          <GuidedFormItem name="contentYml" label="YAML" rules={[{ required: true }]}><Input.TextArea rows={8} /></GuidedFormItem>
        </Form>
      </Modal>
    </Row>
  );
};

export default AutomationPage;
