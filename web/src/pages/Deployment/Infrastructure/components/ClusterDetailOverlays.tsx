import React from 'react';
import { Button, Card, Descriptions, Drawer, Form, Input, InputNumber, Modal, Select, Space, Tag, Typography } from 'antd';
import type { FormInstance } from 'antd';
import type { ClusterNode, ClusterOperationApproval, IngressInfo, ServiceInfo } from '../../../../api/modules/cluster';
import { GuidedFormItem } from '../../../../components/FormGuidance';

const { Text } = Typography;

type PendingServiceModalState = {
  mode: 'create' | 'edit';
  record?: ServiceInfo;
} | null;

type PendingIngressModalState = {
  mode: 'create' | 'edit';
  record?: IngressInfo;
} | null;

type PendingApprovalOperationState = {
  title: string;
  loadingKey: string;
  approval?: ClusterOperationApproval;
} | null;

interface ClusterDetailOverlaysProps {
  serviceModalVisible: boolean;
  pendingServiceModal: PendingServiceModalState;
  submitServiceModal: () => void;
  onCloseServiceModal: () => void;
  serviceForm: FormInstance;
  ingressModalVisible: boolean;
  pendingIngressModal: PendingIngressModalState;
  submitIngressModal: () => void;
  onCloseIngressModal: () => void;
  ingressForm: FormInstance;
  approvalModalVisible: boolean;
  pendingApprovalOperation: PendingApprovalOperationState;
  submitApprovalToken: () => void;
  onCloseApprovalModal: () => void;
  approvalForm: FormInstance;
  nodeMutationLoadingKey: string;
  nodeDrawerVisible: boolean;
  onCloseNodeDrawer: () => void;
  selectedNode: ClusterNode | null;
  getNodeStatusBadge: (status: string) => React.ReactNode;
  nodeLabelForm: FormInstance;
  nodeTaintForm: FormInstance;
  nodeMetadataLoadingKey: string;
  handleNodeMetadataOperation: (
    kind: 'label' | 'taint',
    mode: 'upsert' | 'remove',
    node: ClusterNode,
    values: { key: string; value?: string; effect?: string; approvalToken?: string },
  ) => Promise<void> | void;
}

const ClusterDetailOverlays: React.FC<ClusterDetailOverlaysProps> = ({
  serviceModalVisible,
  pendingServiceModal,
  submitServiceModal,
  onCloseServiceModal,
  serviceForm,
  ingressModalVisible,
  pendingIngressModal,
  submitIngressModal,
  onCloseIngressModal,
  ingressForm,
  approvalModalVisible,
  pendingApprovalOperation,
  submitApprovalToken,
  onCloseApprovalModal,
  approvalForm,
  nodeMutationLoadingKey,
  nodeDrawerVisible,
  onCloseNodeDrawer,
  selectedNode,
  getNodeStatusBadge,
  nodeLabelForm,
  nodeTaintForm,
  nodeMetadataLoadingKey,
  handleNodeMetadataOperation,
}) => (
  <>
    <Modal
      title={pendingServiceModal?.mode === 'edit' ? '编辑 Service' : '新建 Service'}
      open={serviceModalVisible}
      onCancel={onCloseServiceModal}
      onOk={submitServiceModal}
      okText="保存 Service"
      cancelText="取消"
      confirmLoading={Boolean(
        pendingServiceModal
        && nodeMutationLoadingKey === `service:${pendingServiceModal.record?.name || serviceForm.getFieldValue('name')}:${pendingServiceModal.mode}`,
      )}
      destroyOnHidden
    >
      <Form form={serviceForm} layout="vertical">
        <GuidedFormItem name="name" label="service_name" rules={[{ required: true, message: '请输入 Service 名称' }]}>
          <Input aria-label="service_name" disabled={pendingServiceModal?.mode === 'edit'} />
        </GuidedFormItem>
        <Form.Item name="type" label="service_type" rules={[{ required: true, message: '请选择 Service 类型' }]}>
          <Select
            aria-label="service_type"
            options={[
              { label: 'ClusterIP', value: 'ClusterIP' },
              { label: 'NodePort', value: 'NodePort' },
              { label: 'LoadBalancer', value: 'LoadBalancer' },
            ]}
          />
        </Form.Item>
        <GuidedFormItem name="selector_text" label="selector" rules={[{ required: true, message: '请输入 selector，格式 key=value' }]}>
          <Input aria-label="selector" placeholder="app=web,component=api" />
        </GuidedFormItem>
        <GuidedFormItem name="port" label="service_port" rules={[{ required: true, message: '请输入端口' }]}>
          <InputNumber min={1} max={65535} className="w-full" aria-label="service_port" />
        </GuidedFormItem>
        <GuidedFormItem name="target_port" label="target_port" rules={[{ required: true, message: '请输入 target_port' }]}>
          <Input aria-label="target_port" />
        </GuidedFormItem>
        <Form.Item name="protocol" label="protocol" initialValue="TCP">
          <Select aria-label="protocol" options={[{ label: 'TCP', value: 'TCP' }, { label: 'UDP', value: 'UDP' }]} />
        </Form.Item>
        <GuidedFormItem name="node_port" label="node_port">
          <InputNumber min={30000} max={32767} className="w-full" aria-label="node_port" />
        </GuidedFormItem>
      </Form>
    </Modal>

    <Modal
      title={pendingIngressModal?.mode === 'edit' ? '编辑 Ingress' : '新建 Ingress'}
      open={ingressModalVisible}
      onCancel={onCloseIngressModal}
      onOk={submitIngressModal}
      okText="保存 Ingress"
      cancelText="取消"
      confirmLoading={Boolean(
        pendingIngressModal
        && nodeMutationLoadingKey === `ingress:${pendingIngressModal.record?.name || ingressForm.getFieldValue('name')}:${pendingIngressModal.mode}`,
      )}
      destroyOnHidden
    >
      <Form form={ingressForm} layout="vertical">
        <GuidedFormItem name="name" label="ingress_name" rules={[{ required: true, message: '请输入 Ingress 名称' }]}>
          <Input aria-label="ingress_name" disabled={pendingIngressModal?.mode === 'edit'} />
        </GuidedFormItem>
        <GuidedFormItem name="ingress_class_name" label="ingress_class_name">
          <Input aria-label="ingress_class_name" />
        </GuidedFormItem>
        <GuidedFormItem name="host" label="ingress_host" rules={[{ required: true, message: '请输入主机名' }]}>
          <Input aria-label="ingress_host" />
        </GuidedFormItem>
        <GuidedFormItem name="path" label="ingress_path" rules={[{ required: true, message: '请输入路径' }]}>
          <Input aria-label="ingress_path" />
        </GuidedFormItem>
        <Form.Item name="path_type" label="path_type" initialValue="Prefix">
          <Select
            aria-label="path_type"
            options={[
              { label: 'Prefix', value: 'Prefix' },
              { label: 'Exact', value: 'Exact' },
              { label: 'ImplementationSpecific', value: 'ImplementationSpecific' },
            ]}
          />
        </Form.Item>
        <GuidedFormItem name="service_name" label="backend_service_name" rules={[{ required: true, message: '请输入后端 Service 名称' }]}>
          <Input aria-label="backend_service_name" />
        </GuidedFormItem>
        <GuidedFormItem name="service_port" label="backend_service_port" rules={[{ required: true, message: '请输入后端 Service 端口' }]}>
          <InputNumber min={1} max={65535} className="w-full" aria-label="backend_service_port" />
        </GuidedFormItem>
        <GuidedFormItem name="tls_secret_name" label="tls_secret_name">
          <Input aria-label="tls_secret_name" />
        </GuidedFormItem>
      </Form>
    </Modal>

    <Modal
      title="审批确认"
      open={approvalModalVisible}
      onCancel={onCloseApprovalModal}
      onOk={submitApprovalToken}
      okText="提交审批"
      cancelText="取消"
      confirmLoading={Boolean(pendingApprovalOperation && nodeMutationLoadingKey === pendingApprovalOperation.loadingKey)}
      destroyOnHidden
    >
      <Space direction="vertical" className="w-full" size={12}>
        {pendingApprovalOperation?.approval?.ticket && (
          <Text>
            审批单号: <Text code>{pendingApprovalOperation.approval.ticket}</Text>
          </Text>
        )}
        <Text type="secondary">
          {pendingApprovalOperation?.title ? `为“${pendingApprovalOperation.title}”提交审批 token 后继续执行。` : '请输入 approval_token 以继续执行。'}
        </Text>
        <Form form={approvalForm} layout="vertical">
          <GuidedFormItem
            name="approval_token"
            label="approval_token"
            rules={[{ required: true, message: '请输入 approval_token' }]}
          >
            <Input placeholder="请输入审批 token" autoComplete="off" />
          </GuidedFormItem>
        </Form>
      </Space>
    </Modal>

    <Drawer title={`节点详情: ${selectedNode?.name}`} placement="right" width={600} onClose={onCloseNodeDrawer} open={nodeDrawerVisible}>
      {selectedNode && (
        <Space direction="vertical" className="w-full" size={16}>
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="名称" span={2}>{selectedNode.name}</Descriptions.Item>
            <Descriptions.Item label="IP">{selectedNode.ip}</Descriptions.Item>
            <Descriptions.Item label="状态">{getNodeStatusBadge(selectedNode.status)}</Descriptions.Item>
            <Descriptions.Item label="角色"><Tag color={selectedNode.role === 'control-plane' ? 'blue' : 'green'}>{selectedNode.role}</Tag></Descriptions.Item>
            <Descriptions.Item label="Kubelet">{selectedNode.kubelet_version}</Descriptions.Item>
            <Descriptions.Item label="容器运行时">{selectedNode.container_runtime}</Descriptions.Item>
            <Descriptions.Item label="操作系统">{selectedNode.os_image}</Descriptions.Item>
            <Descriptions.Item label="内核版本">{selectedNode.kernel_version}</Descriptions.Item>
            <Descriptions.Item label="CPU">{selectedNode.allocatable_cpu}</Descriptions.Item>
            <Descriptions.Item label="内存">{selectedNode.allocatable_mem}</Descriptions.Item>
          </Descriptions>

          <Card size="small" title="标签">
            <Space wrap className="mb-3">
              {selectedNode.labels && Object.keys(selectedNode.labels).length > 0 ? (
                Object.entries(selectedNode.labels).map(([key, value]) => (
                  <Tag key={key} color="blue">{key}={value}</Tag>
                ))
              ) : <Text type="secondary">暂无标签</Text>}
            </Space>
            <Form
              form={nodeLabelForm}
              layout="vertical"
              onFinish={async (values: { key: string; value?: string }) => {
                await handleNodeMetadataOperation('label', 'upsert', selectedNode, values);
              }}
            >
              <div className="grid grid-cols-1 gap-3">
                <GuidedFormItem name="key" label="标签键" rules={[{ required: true }]} className="mb-0">
                  <Input placeholder="app.kubernetes.io/name" />
                </GuidedFormItem>
                <GuidedFormItem name="value" label="标签值" className="mb-0">
                  <Input placeholder="frontend" />
                </GuidedFormItem>
              </div>
              <Space className="mt-3">
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={nodeMetadataLoadingKey.startsWith(`${selectedNode.name}:label`)}
                >
                  保存标签
                </Button>
                <Button
                  danger
                  onClick={async () => {
                    const values = await nodeLabelForm.validateFields();
                    await handleNodeMetadataOperation('label', 'remove', selectedNode, values);
                  }}
                  loading={nodeMetadataLoadingKey.startsWith(`${selectedNode.name}:label`)}
                >
                  删除标签
                </Button>
              </Space>
            </Form>
          </Card>

          <Card size="small" title="污点">
            <Space wrap className="mb-3">
              {selectedNode.taints && selectedNode.taints.length > 0 ? (
                selectedNode.taints.map((taint) => (
                  <Tag key={`${taint.key}:${taint.effect}`} color="orange">
                    {taint.key}={taint.value || '-'}:{taint.effect}
                  </Tag>
                ))
              ) : <Text type="secondary">暂无污点</Text>}
            </Space>
            <Form
              form={nodeTaintForm}
              layout="vertical"
              onFinish={async (values: { key: string; value?: string; effect?: string }) => {
                await handleNodeMetadataOperation('taint', 'upsert', selectedNode, values);
              }}
            >
              <div className="grid grid-cols-1 gap-3">
                <GuidedFormItem name="key" label="污点键" rules={[{ required: true }]} className="mb-0">
                  <Input placeholder="node-role.kubernetes.io/worker" />
                </GuidedFormItem>
                <GuidedFormItem name="value" label="污点值" className="mb-0">
                  <Input placeholder="value" />
                </GuidedFormItem>
                <Form.Item name="effect" label="效果" className="mb-0">
                  <Select
                    options={[
                      { value: 'NoSchedule', label: 'NoSchedule' },
                      { value: 'PreferNoSchedule', label: 'PreferNoSchedule' },
                      { value: 'NoExecute', label: 'NoExecute' },
                    ]}
                  />
                </Form.Item>
              </div>
              <Space className="mt-3">
                <Button
                  type="primary"
                  htmlType="submit"
                  loading={nodeMetadataLoadingKey.startsWith(`${selectedNode.name}:taint`)}
                >
                  保存污点
                </Button>
                <Button
                  danger
                  onClick={async () => {
                    const values = await nodeTaintForm.validateFields();
                    await handleNodeMetadataOperation('taint', 'remove', selectedNode, values);
                  }}
                  loading={nodeMetadataLoadingKey.startsWith(`${selectedNode.name}:taint`)}
                >
                  删除污点
                </Button>
              </Space>
            </Form>
          </Card>
        </Space>
      )}
    </Drawer>
  </>
);

export default ClusterDetailOverlays;
