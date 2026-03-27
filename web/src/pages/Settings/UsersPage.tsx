import React from 'react';
import { Button, Card, Drawer, Empty, Form, Input, Modal, Select, Space, Table, Tag, message } from 'antd';
import { EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import { Api } from '../../api';
import type { Role, User } from '../../api/modules/rbac';
import { ApiRequestError } from '../../api/api';
import AccessDeniedPage from '../../components/Auth/AccessDeniedPage';
import { usePermission } from '../../components/RBAC/PermissionContext';

const UsersPage: React.FC = () => {
  const { hasPermission } = usePermission();
  const canWrite = hasPermission('rbac', 'write');

  const [loading, setLoading] = React.useState(false);
  const [list, setList] = React.useState<User[]>([]);
  const [roles, setRoles] = React.useState<Role[]>([]);
  const [open, setOpen] = React.useState(false);
  const [editOpen, setEditOpen] = React.useState(false);
  const [query, setQuery] = React.useState('');
  const [active, setActive] = React.useState<User | null>(null);
  const [editingUser, setEditingUser] = React.useState<User | null>(null);
  const [accessDenied, setAccessDenied] = React.useState(false);
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  const roleOptions = React.useMemo(() => {
    if (roles.length === 0) {
      return [{ value: 'admin' }, { value: 'operator' }, { value: 'viewer' }];
    }
    return roles.map((role) => ({
      value: role.code || role.name,
      label: role.name,
    }));
  }, [roles]);

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const [userRes, roleRes] = await Promise.all([
        Api.rbac.getUserList({ page: 1, pageSize: 200 }),
        Api.rbac.getRoleList({ page: 1, pageSize: 200 }),
      ]);
      setList(userRes.data.list || []);
      setRoles(roleRes.data.list || []);
      setAccessDenied(false);
    } catch (err) {
      if (err instanceof ApiRequestError && (err.statusCode === 403 || err.businessCode === 2004)) {
        setAccessDenied(true);
        return;
      }
      message.error(err instanceof Error ? err.message : '加载用户失败');
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  React.useEffect(() => {
    if (!editingUser) {
      editForm.resetFields();
      return;
    }
    editForm.setFieldsValue({
      email: editingUser.email,
      roles: editingUser.roles,
      status: editingUser.status || 'active',
      password: '',
    });
  }, [editingUser, editForm]);

  const create = async () => {
    if (!canWrite) return;
    const values = await form.validateFields();
    const startedAt = performance.now();
    await Api.rbac.createUser({
      username: values.username,
      name: values.username,
      email: values.email,
      password: values.password,
      roles: values.roles,
      status: values.status || 'active',
    });
    void Api.rbac.recordMigrationEvent({
      eventType: 'governance_task',
      action: 'user.create',
      status: 'success',
      durationMs: Math.round(performance.now() - startedAt),
    }).catch(() => undefined);
    message.success('创建成功');
    setOpen(false);
    form.resetFields();
    void load();
  };

  const submitEdit = async () => {
    if (!canWrite || !editingUser) return;
    const values = await editForm.validateFields();

    Modal.confirm({
      title: '确认更新用户',
      content: `将更新用户「${editingUser.username}」的账号信息和角色绑定，确认继续？`,
      okText: '确认更新',
      onOk: async () => {
        const startedAt = performance.now();
        await Api.rbac.updateUser(editingUser.id, {
          email: values.email,
          roles: values.roles,
          status: values.status,
          password: values.password || undefined,
        });
        void Api.rbac.recordMigrationEvent({
          eventType: 'governance_task',
          action: 'user.update',
          status: 'success',
          durationMs: Math.round(performance.now() - startedAt),
        }).catch(() => undefined);
        message.success('用户更新成功');
        setEditOpen(false);
        setEditingUser(null);
        void load();
      },
    });
  };

  const remove = async (user: User) => {
    if (!canWrite) return;
    Modal.confirm({
      title: '删除用户',
      content: `将删除用户「${user.username}」，本次操作影响 1 个用户账号，确认继续？`,
      okText: '确认删除',
      okButtonProps: { danger: true },
      onOk: async () => {
        const startedAt = performance.now();
        await Api.rbac.deleteUser(user.id);
        void Api.rbac.recordMigrationEvent({
          eventType: 'governance_task',
          action: 'user.delete',
          status: 'success',
          durationMs: Math.round(performance.now() - startedAt),
        }).catch(() => undefined);
        message.success('删除成功');
        if (active?.id === user.id) {
          setActive(null);
        }
        void load();
      },
    });
  };

  const filtered = list.filter((item) => {
    const q = query.trim().toLowerCase();
    if (!q) return true;
    return [item.username, item.name, item.email].some((value) => String(value || '').toLowerCase().includes(q));
  });

  if (accessDenied) {
    return <AccessDeniedPage />;
  }

  return (
    <Card
      title="用户管理"
      extra={(
        <Space>
          <Input
            allowClear
            aria-label="搜索用户"
            placeholder="搜索用户名/姓名/邮箱"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            style={{ width: 260 }}
          />
          <Button className="governance-action-btn" icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>刷新</Button>
          {canWrite ? (
            <Button className="governance-action-btn" type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>新增用户</Button>
          ) : null}
        </Space>
      )}
    >
      <Table
        rowKey="id"
        loading={loading}
        locale={{ emptyText: <Empty description="暂无用户数据" /> }}
        dataSource={filtered}
        onRow={(record) => ({
          onClick: () => setActive(record),
          onKeyDown: (event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault();
              setActive(record);
            }
          },
          tabIndex: 0,
          role: 'button',
          className: 'governance-interactive-row',
          'aria-label': `查看用户 ${record.username} 详情`,
        })}
        columns={[
          { title: '用户名', dataIndex: 'username' },
          { title: '姓名', dataIndex: 'name' },
          { title: '邮箱', dataIndex: 'email' },
          {
            title: '角色',
            dataIndex: 'roles',
            render: (items: string[]) => <>{items?.map((role) => <Tag key={role}>{role}</Tag>)}</>,
          },
          {
            title: '状态',
            dataIndex: 'status',
            render: (value: string) => <Tag color={value === 'active' ? 'success' : 'default'}>{value}</Tag>,
          },
          {
            title: '操作',
            width: 260,
            render: (_: unknown, row: User) => (
              <Space>
                <Button
                  type="link"
                  className="governance-action-btn"
                  aria-label={`查看用户 ${row.username} 详情`}
                  onClick={(event) => {
                    event.stopPropagation();
                    setActive(row);
                  }}
                >
                  详情
                </Button>
                {canWrite ? (
                  <Button
                    type="link"
                    icon={<EditOutlined />}
                    className="governance-action-btn"
                    aria-label={`编辑用户 ${row.username}`}
                    onClick={(event) => {
                      event.stopPropagation();
                      setEditingUser(row);
                      setEditOpen(true);
                    }}
                  >
                    编辑
                  </Button>
                ) : null}
                {canWrite ? (
                  <Button
                    className="governance-action-btn"
                    danger
                    type="link"
                    aria-label={`删除用户 ${row.username}`}
                    onClick={(event) => {
                      event.stopPropagation();
                      void remove(row);
                    }}
                  >
                    删除
                  </Button>
                ) : null}
              </Space>
            ),
          },
        ]}
      />

      <Modal title="新增用户" open={open} onCancel={() => setOpen(false)} onOk={() => void create()} okButtonProps={{ disabled: !canWrite }}>
        <Form form={form} layout="vertical" initialValues={{ roles: ['viewer'], status: 'active' }}>
          <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}><Input /></Form.Item>
          <Form.Item name="email" label="邮箱" rules={[{ required: true, type: 'email', message: '请输入正确邮箱' }]}><Input /></Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true, min: 6, message: '密码至少 6 位' }]}><Input.Password /></Form.Item>
          <Form.Item name="roles" label="角色" rules={[{ required: true, message: '请至少选择一个角色' }]}>
            <Select mode="multiple" options={roleOptions} />
          </Form.Item>
          <Form.Item name="status" label="状态"><Select options={[{ value: 'active' }, { value: 'disabled' }]} /></Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingUser ? `编辑用户：${editingUser.username}` : '编辑用户'}
        open={editOpen}
        onCancel={() => {
          setEditOpen(false);
          setEditingUser(null);
        }}
        onOk={() => void submitEdit()}
        okButtonProps={{ disabled: !canWrite }}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item label="用户名">
            <Input value={editingUser?.username} disabled aria-label="编辑用户用户名" />
          </Form.Item>
          <Form.Item name="email" label="邮箱" rules={[{ required: true, type: 'email', message: '请输入正确邮箱' }]}>
            <Input aria-label="编辑用户邮箱" />
          </Form.Item>
          <Form.Item name="roles" label="角色" rules={[{ required: true, message: '请至少选择一个角色' }]}>
            <Select mode="multiple" options={roleOptions} aria-label="编辑用户角色" />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select options={[{ value: 'active' }, { value: 'disabled' }]} aria-label="编辑用户状态" />
          </Form.Item>
          <Form.Item name="password" label="重置密码（可选）" rules={[{ min: 6, message: '密码至少 6 位' }]}>
            <Input.Password aria-label="编辑用户密码" />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer title="用户详情" open={Boolean(active)} onClose={() => setActive(null)} size="large">
        {!active ? null : (
          <Space orientation="vertical" size={12} style={{ width: '100%' }}>
            <div><strong>用户名：</strong>{active.username}</div>
            <div><strong>姓名：</strong>{active.name}</div>
            <div><strong>邮箱：</strong>{active.email}</div>
            <div>
              <strong>角色：</strong>{' '}
              {(active.roles || []).map((role) => <Tag key={role}>{role}</Tag>)}
            </div>
            <div><strong>状态：</strong>{active.status}</div>
          </Space>
        )}
      </Drawer>
    </Card>
  );
};

export default UsersPage;
