import React, { useEffect, useState, useCallback, useMemo } from 'react';
import {
  Layout,
  Tree,
  Table,
  Button,
  Space,
  Card,
  Modal,
  Form,
  Input,
  InputNumber,
  message,
  Popconfirm,
  Dropdown,
  Tag,
  Empty,
  Typography,
  Tabs,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  UserOutlined,
  TeamOutlined,
  MoreOutlined,
  SwapOutlined,
  SafetyCertificateOutlined,
  KeyOutlined,
} from '@ant-design/icons';
import type { Department, Member, CreateDepartmentParams, UpdateDepartmentParams, TransferMemberParams } from '../../api/modules/org';
import Api from '../../api';
import RolesPage from '../Settings/RolesPage';
import PermissionsPage from '../Settings/PermissionsPage';

const { Sider, Content } = Layout;
const { Text } = Typography;

interface DataNode {
  title: string;
  key: string;
  children?: DataNode[];
  data: Department;
}

const AccessControlPage: React.FC = () => {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedDeptId, setSelectedDeptId] = useState<string | null>(null);
  const [members, setMembers] = useState<Member[]>([]);
  const [membersLoading, setMembersLoading] = useState(false);

  // Department Modals
  const [deptModalOpen, setDeptModalOpen] = useState(false);
  const [deptModalMode, setDeptModalMode] = useState<'create' | 'edit'>('create');
  const [editingDept, setEditingDept] = useState<Department | null>(null);
  const [deptForm] = Form.useForm();

  // Member Transfer Modal
  const [transferModalOpen, setTransferModalOpen] = useState(false);
  const [selectedMemberIds, setSelectedMemberIds] = useState<string[]>([]);
  const [transferForm] = Form.useForm();

  // Role Assignment Modal
  const [roleModalOpen, setRoleModalOpen] = useState(false);
  const [editingMember, setEditingMember] = useState<Member | null>(null);
  const [allRoles, setAllRoles] = useState<any[]>([]);
  const [memberRoles, setMemberRoles] = useState<string[]>([]);

  const fetchDepartments = useCallback(async () => {
    setLoading(true);
    try {
      const res = await Api.org.getDepartmentTree();
      setDepartments(res.data || []);
    } catch (err: any) {
      message.error('获取部门树失败: ' + (err.message || '未知错误'));
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchMembers = useCallback(async (deptId: string) => {
    setMembersLoading(true);
    try {
      const res = await Api.org.getDepartmentMembers(deptId);
      setMembers(res.data || []);
    } catch (err: any) {
      message.error('获取部门成员失败: ' + (err.message || '未知错误'));
    } finally {
      setMembersLoading(false);
    }
  }, []);

  const fetchAllRoles = useCallback(async () => {
    try {
      const res = await Api.rbac.getRoleList({ page: 1, pageSize: 200 });
      setAllRoles(res.data.list || []);
    } catch (err) {
      console.error('Failed to fetch roles');
    }
  }, []);

  useEffect(() => {
    fetchDepartments();
    fetchAllRoles();
  }, [fetchDepartments, fetchAllRoles]);

  useEffect(() => {
    if (selectedDeptId) {
      fetchMembers(selectedDeptId);
    } else {
      setMembers([]);
    }
  }, [selectedDeptId, fetchMembers]);

  const treeData = useMemo(() => {
    const mapTree = (depts: Department[]): DataNode[] => {
      return depts.map((d) => ({
        title: d.name,
        key: d.id,
        children: d.children ? mapTree(d.children) : [],
        data: d,
      }));
    };
    return mapTree(departments);
  }, [departments]);

  const handleCreateDept = (parentId?: string) => {
    setDeptModalMode('create');
    setEditingDept(null);
    deptForm.resetFields();
    deptForm.setFieldsValue({ parentId });
    setDeptModalOpen(true);
  };

  const handleEditDept = (dept: Department) => {
    setDeptModalMode('edit');
    setEditingDept(dept);
    deptForm.setFieldsValue({
      name: dept.name,
      parentId: dept.parentId,
      order: dept.order,
    });
    setDeptModalOpen(true);
  };

  const handleDeleteDept = async (id: string) => {
    try {
      await Api.org.deleteDepartment(id);
      message.success('删除部门成功');
      fetchDepartments();
      if (selectedDeptId === id) setSelectedDeptId(null);
    } catch (err: any) {
      message.error('删除部门失败: ' + (err.message || '未知错误'));
    }
  };

  const handleDeptModalOk = async () => {
    try {
      const values = await deptForm.validateFields();
      if (deptModalMode === 'create') {
        await Api.org.createDepartment(values as CreateDepartmentParams);
        message.success('创建部门成功');
      } else if (editingDept) {
        await Api.org.updateDepartment(editingDept.id, values as UpdateDepartmentParams);
        message.success('更新部门成功');
      }
      setDeptModalOpen(false);
      fetchDepartments();
    } catch (err: any) {
      if (err.errorFields) return;
      message.error('保存部门失败: ' + (err.message || '未知错误'));
    }
  };

  const handleTransferMembers = async () => {
    try {
      const values = await transferForm.validateFields();
      const params: TransferMemberParams = {
        memberIds: selectedMemberIds,
        targetDepartmentId: values.targetDepartmentId,
      };
      await Api.org.transferMember(params);
      message.success('划转成员成功');
      setTransferModalOpen(false);
      setSelectedMemberIds([]);
      if (selectedDeptId) fetchMembers(selectedDeptId);
    } catch (err: any) {
      if (err.errorFields) return;
      message.error('划转成员失败: ' + (err.message || '未知错误'));
    }
  };

  const handleAssignRole = (member: Member) => {
    setEditingMember(member);
    setMemberRoles(member.roles || []);
    setRoleModalOpen(true);
  };

  const handleRoleModalOk = async () => {
    if (!editingMember) return;
    try {
      await Api.rbac.updateUserRoles(editingMember.id, memberRoles);
      message.success('角色分配成功');
      setRoleModalOpen(false);
      if (selectedDeptId) fetchMembers(selectedDeptId);
    } catch (err: any) {
      message.error('角色分配失败: ' + (err.message || '未知错误'));
    }
  };

  const memberColumns = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      render: (text: string) => <Space><UserOutlined />{text}</Space>,
    },
    {
      title: '姓名',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: '角色',
      dataIndex: 'roles',
      key: 'roles',
      render: (roles: string[]) => (
        <Space wrap>
          {(roles || []).map(r => <Tag key={r} color="blue">{r}</Tag>)}
          {(!roles || roles.length === 0) && <Text type="secondary">未分配</Text>}
        </Space>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'success' : 'default'}>
          {status === 'active' ? '活跃' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Member) => (
        <Button 
          type="link" 
          size="small" 
          icon={<SafetyCertificateOutlined />} 
          onClick={() => handleAssignRole(record)}
        >
          分配角色
        </Button>
      ),
    },
  ];

  const renderOrgTab = () => (
    <Layout style={{ background: '#fff', minHeight: '600px' }}>
      <Sider width={280} theme="light" style={{ borderRight: '1px solid #f0f0f0', padding: '0 16px 0 0' }}>
        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Text strong>部门架构</Text>
          <Button
            type="primary"
            size="small"
            icon={<PlusOutlined />}
            onClick={() => handleCreateDept()}
          >
            新增
          </Button>
        </div>
        {departments.length > 0 ? (
          <Tree
            showLine
            switcherIcon={<TeamOutlined />}
            treeData={treeData}
            onSelect={(keys) => setSelectedDeptId(keys[0] as string)}
            selectedKeys={selectedDeptId ? [selectedDeptId] : []}
            titleRender={(node: any) => (
              <div style={{ display: 'flex', justifyContent: 'space-between', width: '180px' }}>
                <span className="truncate">{node.title}</span>
                <Dropdown
                  menu={{
                    items: [
                      {
                        key: 'add',
                        label: '子部门',
                        icon: <PlusOutlined />,
                        onClick: () => handleCreateDept(node.key),
                      },
                      {
                        key: 'edit',
                        label: '编辑',
                        icon: <EditOutlined />,
                        onClick: () => handleEditDept(node.data),
                      },
                      {
                        key: 'delete',
                        label: '删除',
                        danger: true,
                        icon: <DeleteOutlined />,
                        onClick: () => {
                          Modal.confirm({
                            title: '确定删除该部门吗？',
                            onOk: () => handleDeleteDept(node.key),
                          });
                        },
                      },
                    ],
                  }}
                  trigger={['click']}
                >
                  <MoreOutlined onClick={(e) => e.stopPropagation()} />
                </Dropdown>
              </div>
            )}
          />
        ) : (
          <Empty description="暂无部门" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
      </Sider>
      <Content style={{ padding: '0 0 0 24px' }}>
        <Card
          size="small"
          title={selectedDeptId ? `成员列表 - ${departments.find(d => d.id === selectedDeptId)?.name || ''}` : '成员列表'}
          extra={
            <Space>
              <Button
                size="small"
                icon={<SwapOutlined />}
                disabled={selectedMemberIds.length === 0}
                onClick={() => {
                  transferForm.resetFields();
                  setTransferModalOpen(true);
                }}
              >
                批量划转
              </Button>
            </Space>
          }
        >
          <Table
            rowSelection={{
              type: 'checkbox',
              selectedRowKeys: selectedMemberIds,
              onChange: (keys) => setSelectedMemberIds(keys as string[]),
            }}
            columns={memberColumns}
            dataSource={members}
            rowKey="id"
            loading={membersLoading}
            size="small"
            pagination={{ pageSize: 10 }}
            locale={{ emptyText: selectedDeptId ? '该部门暂无成员' : '请先选择部门' }}
          />
        </Card>
      </Content>
    </Layout>
  );

  const tabItems = [
    {
      key: 'org',
      label: <Space><TeamOutlined />组织与成员</Space>,
      children: renderOrgTab(),
    },
    {
      key: 'roles',
      label: <Space><SafetyCertificateOutlined />角色管理</Space>,
      children: <RolesPage />,
    },
    {
      key: 'permissions',
      label: <Space><KeyOutlined />权限定义</Space>,
      children: <PermissionsPage />,
    },
  ];

  return (
    <div style={{ padding: '24px', background: '#fff', minHeight: 'calc(100vh - 64px)' }}>
      <div style={{ marginBottom: 24 }}>
        <Typography.Title level={4}>访问控制中心</Typography.Title>
        <Text type="secondary">统一管理系统组织架构、成员身份及访问权限。</Text>
      </div>
      
      <Tabs defaultActiveKey="org" items={tabItems} />

      {/* Department Modal */}
      <Modal
        title={deptModalMode === 'create' ? '新增部门' : '编辑部门'}
        open={deptModalOpen}
        onOk={handleDeptModalOk}
        onCancel={() => setDeptModalOpen(false)}
        destroyOnHidden
      >
        <Form form={deptForm} layout="vertical">
          <Form.Item name="name" label="部门名称" rules={[{ required: true, message: '请输入部门名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="parentId" label="上级部门">
            <Tree
              treeData={treeData}
              onSelect={(keys) => deptForm.setFieldsValue({ parentId: keys[0] })}
              style={{ border: '1px solid #d9d9d9', padding: '8px', borderRadius: '4px', maxHeight: '200px', overflow: 'auto' }}
            />
          </Form.Item>
          <Form.Item name="order" label="排序权重" initialValue={0}>
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Transfer Modal */}
      <Modal
        title="批量划转成员"
        open={transferModalOpen}
        onOk={handleTransferMembers}
        onCancel={() => setTransferModalOpen(false)}
        destroyOnHidden
      >
        <Form form={transferForm} layout="vertical">
          <div style={{ marginBottom: 16 }}>
            已选择 <Text type="danger">{selectedMemberIds.length}</Text> 名成员
          </div>
          <Form.Item name="targetDepartmentId" label="目标部门" rules={[{ required: true, message: '请选择目标部门' }]}>
            <Tree
              treeData={treeData}
              onSelect={(keys) => transferForm.setFieldsValue({ targetDepartmentId: keys[0] })}
              style={{ border: '1px solid #d9d9d9', padding: '8px', borderRadius: '4px', maxHeight: '200px', overflow: 'auto' }}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Role Assignment Modal */}
      <Modal
        title={`分配角色 - ${editingMember?.username || ''}`}
        open={roleModalOpen}
        onOk={handleRoleModalOk}
        onCancel={() => setRoleModalOpen(false)}
        destroyOnHidden
      >
        <div style={{ marginBottom: 16 }}>
          请选择授予该成员的系统角色：
        </div>
        <div style={{ maxHeight: '300px', overflow: 'auto' }}>
          <Table
            size="small"
            rowSelection={{
              type: 'checkbox',
              selectedRowKeys: memberRoles,
              onChange: (keys) => setMemberRoles(keys as string[]),
            }}
            columns={[{ title: '角色名', dataIndex: 'name' }, { title: '描述', dataIndex: 'description' }]}
            dataSource={allRoles}
            rowKey="name"
            pagination={false}
          />
        </div>
      </Modal>
    </div>
  );
};

export default AccessControlPage;
