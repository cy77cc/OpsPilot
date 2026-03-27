import React from 'react';
import { Button, Card, Drawer, Empty, Form, Input, Modal, Space, Table, Tag, Tree, Typography, message } from 'antd';
import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from '@ant-design/icons';
import type { DataNode } from 'antd/es/tree';
import { Api } from '../../api';
import type { Permission, Role } from '../../api/modules/rbac';
import { ApiRequestError } from '../../api/api';
import AccessDeniedPage from '../../components/Auth/AccessDeniedPage';
import { usePermission } from '../../components/RBAC/PermissionContext';
import {
  filterPermissions,
  getFilteredPermissionCodes,
  groupPermissions,
  inverseSelection,
  summarizePermissionChanges,
} from './rbacPermissionUtils';

const { Text } = Typography;

const RolesPage: React.FC = () => {
  const { hasPermission } = usePermission();
  const canWrite = hasPermission('rbac', 'write');

  const [loading, setLoading] = React.useState(false);
  const [roles, setRoles] = React.useState<Role[]>([]);
  const [permissions, setPermissions] = React.useState<Permission[]>([]);
  const [open, setOpen] = React.useState(false);
  const [query, setQuery] = React.useState('');
  const [active, setActive] = React.useState<Role | null>(null);
  const [accessDenied, setAccessDenied] = React.useState(false);
  const [permissionQuery, setPermissionQuery] = React.useState('');
  const [editingPermissions, setEditingPermissions] = React.useState<string[]>([]);

  const [detailForm] = Form.useForm();
  const [form] = Form.useForm();

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const [roleRes, permRes] = await Promise.all([
        Api.rbac.getRoleList({ page: 1, pageSize: 200 }),
        Api.rbac.getPermissionList({ page: 1, pageSize: 1000 }),
      ]);
      setRoles(roleRes.data.list || []);
      setPermissions(permRes.data.list || []);
      setAccessDenied(false);
    } catch (err) {
      if (err instanceof ApiRequestError && (err.statusCode === 403 || err.businessCode === 2004)) {
        setAccessDenied(true);
        return;
      }
      message.error(err instanceof Error ? err.message : '加载角色失败');
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  const create = async () => {
    if (!canWrite) return;
    const values = await form.validateFields();
    const startedAt = performance.now();
    await Api.rbac.createRole({
      name: values.name,
      description: values.description,
      permissions: values.permissions || [],
    });
    void Api.rbac.recordMigrationEvent({
      eventType: 'governance_task',
      action: 'role.create',
      status: 'success',
      durationMs: Math.round(performance.now() - startedAt),
    }).catch(() => undefined);
    message.success('创建角色成功');
    setOpen(false);
    form.resetFields();
    void load();
  };

  const deleteRole = async (role: Role) => {
    if (!canWrite) return;
    Modal.confirm({
      title: '删除角色',
      content: `将删除角色「${role.name}」，并解除关联授权。确认继续？`,
      okText: '确认删除',
      okButtonProps: { danger: true },
      onOk: async () => {
        const startedAt = performance.now();
        await Api.rbac.deleteRole(role.id);
        void Api.rbac.recordMigrationEvent({
          eventType: 'governance_task',
          action: 'role.delete',
          status: 'success',
          durationMs: Math.round(performance.now() - startedAt),
        }).catch(() => undefined);
        message.success('角色删除成功');
        if (active?.id === role.id) {
          setActive(null);
        }
        void load();
      },
    });
  };

  const openRoleDetail = (role: Role) => {
    setActive(role);
    setPermissionQuery('');
  };

  const filteredRoles = roles.filter((item) => {
    const q = query.trim().toLowerCase();
    if (!q) return true;
    return [item.name, item.description].some((v) => String(v || '').toLowerCase().includes(q));
  });

  React.useEffect(() => {
    if (!active) {
      detailForm.resetFields();
      setEditingPermissions([]);
      return;
    }
    detailForm.setFieldsValue({
      name: active.name,
      description: active.description,
    });
    setEditingPermissions(active.permissions || []);
  }, [active, detailForm]);

  const groupedPermissions = React.useMemo(() => {
    return groupPermissions(filterPermissions(permissions, permissionQuery));
  }, [permissions, permissionQuery]);

  const filteredPermissionCodes = React.useMemo(() => {
    return getFilteredPermissionCodes(permissions, permissionQuery);
  }, [permissions, permissionQuery]);

  const treeData = React.useMemo<DataNode[]>(() => {
    return groupedPermissions.map((group) => ({
      key: `group:${group.key}`,
      title: `${group.label} (${group.permissions.length})`,
      children: group.permissions.map((permission) => ({
        key: permission.code,
        title: `${permission.code} (${permission.name})`,
      })),
    }));
  }, [groupedPermissions]);

  const applyPermissionSelection = (next: string[]) => {
    const deduped = Array.from(new Set(next));
    setEditingPermissions(deduped);
  };

  const selectFiltered = () => {
    applyPermissionSelection([...editingPermissions, ...filteredPermissionCodes]);
  };

  const clearFiltered = () => {
    const scope = new Set(filteredPermissionCodes);
    applyPermissionSelection(editingPermissions.filter((code) => !scope.has(code)));
  };

  const inverseFiltered = () => {
    applyPermissionSelection(inverseSelection(editingPermissions, filteredPermissionCodes));
  };

  const clearAll = () => {
    applyPermissionSelection([]);
  };

  const saveRolePermissions = async () => {
    if (!active || !canWrite) return;
    const values = await detailForm.validateFields();
    const summary = summarizePermissionChanges(active.permissions || [], editingPermissions);

    Modal.confirm({
      title: '确认更新角色权限',
      content: `本次新增 ${summary.added} 项、移除 ${summary.removed} 项权限，影响对象：1 个角色。是否继续？`,
      okText: '确认更新',
      onOk: async () => {
        const startedAt = performance.now();
        await Api.rbac.updateRole(active.id, {
          name: values.name,
          description: values.description,
          permissions: editingPermissions,
        });
        void Api.rbac.recordMigrationEvent({
          eventType: 'governance_task',
          action: 'role.update_permissions',
          status: 'success',
          durationMs: Math.round(performance.now() - startedAt),
        }).catch(() => undefined);
        message.success('角色权限更新成功');
        setActive(null);
        void load();
      },
    });
  };

  if (accessDenied) {
    return <AccessDeniedPage />;
  }

  return (
    <Card
      title="角色管理"
      extra={(
        <Space>
          <Input
            allowClear
            aria-label="搜索角色"
            placeholder="搜索角色名/描述"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            style={{ width: 260 }}
          />
          <Button className="governance-action-btn" icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>刷新</Button>
          {canWrite ? (
            <Button className="governance-action-btn" type="primary" icon={<PlusOutlined />} onClick={() => setOpen(true)}>新增角色</Button>
          ) : null}
        </Space>
      )}
    >
      <Table
        rowKey="id"
        loading={loading}
        locale={{ emptyText: <Empty description="暂无角色数据" /> }}
        dataSource={filteredRoles}
        onRow={(record) => ({
          onClick: () => openRoleDetail(record),
          onKeyDown: (event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault();
              openRoleDetail(record);
            }
          },
          tabIndex: 0,
          role: 'button',
          className: 'governance-interactive-row',
          'aria-label': `查看角色 ${record.name} 详情`,
        })}
        columns={[
          { title: '角色', dataIndex: 'name' },
          { title: '描述', dataIndex: 'description' },
          {
            title: '权限',
            dataIndex: 'permissions',
            render: (perms: string[]) => (
              <Space wrap>
                {(perms || []).slice(0, 6).map((code) => <Tag key={code}>{code}</Tag>)}
                {(perms || []).length > 6 ? <Tag>+{(perms || []).length - 6}</Tag> : null}
              </Space>
            ),
          },
          {
            title: '操作',
            width: 260,
            render: (_: unknown, row: Role) => (
              <Space>
                <Button
                  type="link"
                  className="governance-action-btn"
                  aria-label={`查看角色 ${row.name} 详情`}
                  onClick={(event) => {
                    event.stopPropagation();
                    openRoleDetail(row);
                  }}
                >
                  详情
                </Button>
                {canWrite ? (
                  <Button
                    type="link"
                    icon={<EditOutlined />}
                    className="governance-action-btn"
                    aria-label={`编辑角色 ${row.name} 权限`}
                    onClick={(event) => {
                      event.stopPropagation();
                      openRoleDetail(row);
                    }}
                  >
                    编辑权限
                  </Button>
                ) : null}
                {canWrite ? (
                  <Button
                    type="link"
                    danger
                    icon={<DeleteOutlined />}
                    className="governance-action-btn"
                    aria-label={`删除角色 ${row.name}`}
                    onClick={(event) => {
                      event.stopPropagation();
                      void deleteRole(row);
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

      <Modal title="新增角色" open={open} onCancel={() => setOpen(false)} onOk={() => void create()} width={760} okButtonProps={{ disabled: !canWrite }}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="角色名" rules={[{ required: true, message: '请输入角色名' }]}><Input /></Form.Item>
          <Form.Item name="description" label="描述"><Input /></Form.Item>
          <Form.Item name="permissions" label="权限">
            <Tree
              checkable
              selectable={false}
              height={320}
              treeData={groupPermissions(permissions).map((group) => ({
                key: `create:${group.key}`,
                title: `${group.label} (${group.permissions.length})`,
                children: group.permissions.map((permission) => ({ key: permission.code, title: `${permission.code} (${permission.name})` })),
              }))}
              onCheck={(checked) => {
                const keys = Array.isArray(checked) ? checked : checked.checked;
                form.setFieldValue('permissions', keys.filter((key) => !String(key).startsWith('create:')));
              }}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title="角色详情与权限编辑"
        open={Boolean(active)}
        onClose={() => setActive(null)}
        width={760}
        extra={canWrite ? <Button className="governance-action-btn" type="primary" onClick={() => void saveRolePermissions()}>保存变更</Button> : null}
      >
        <Form form={detailForm} layout="vertical">
          <Form.Item name="name" label="角色名" rules={[{ required: true, message: '请输入角色名' }]}>
            <Input disabled={!canWrite} aria-label="角色名" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input disabled={!canWrite} aria-label="角色描述" />
          </Form.Item>
        </Form>

        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Space wrap>
            <Input
              allowClear
              value={permissionQuery}
              onChange={(event) => setPermissionQuery(event.target.value)}
              style={{ width: 320 }}
              placeholder="搜索权限 code/名称/描述/分组"
              aria-label="搜索角色权限"
              disabled={!canWrite}
            />
            <Button onClick={selectFiltered} disabled={!canWrite || filteredPermissionCodes.length === 0}>全选筛选结果</Button>
            <Button onClick={clearFiltered} disabled={!canWrite || filteredPermissionCodes.length === 0}>清空筛选结果</Button>
            <Button onClick={inverseFiltered} disabled={!canWrite || filteredPermissionCodes.length === 0}>反选筛选结果</Button>
            <Button onClick={clearAll} disabled={!canWrite || editingPermissions.length === 0}>清空全部</Button>
          </Space>

          <Space>
            <Text type="secondary">已选权限：{editingPermissions.length}</Text>
            <Text type="secondary">筛选命中：{filteredPermissionCodes.length}</Text>
          </Space>

          <Tree
            checkable
            selectable={false}
            disabled={!canWrite}
            height={420}
            checkedKeys={editingPermissions}
            treeData={treeData}
            onCheck={(checked) => {
              const keys = Array.isArray(checked) ? checked : checked.checked;
              applyPermissionSelection(keys.filter((key) => !String(key).startsWith('group:')) as string[]);
            }}
          />
        </Space>
      </Drawer>
    </Card>
  );
};

export default RolesPage;
