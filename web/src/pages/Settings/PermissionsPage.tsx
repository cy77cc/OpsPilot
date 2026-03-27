import React from 'react';
import { Button, Card, Empty, Input, Select, Space, Table, Tag, Drawer, message } from 'antd';
import { CopyOutlined, ReloadOutlined } from '@ant-design/icons';
import { Api } from '../../api';
import type { Permission } from '../../api/modules/rbac';
import { ApiRequestError } from '../../api/api';
import AccessDeniedPage from '../../components/Auth/AccessDeniedPage';

const PermissionsPage: React.FC = () => {
  const [list, setList] = React.useState<Permission[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [query, setQuery] = React.useState('');
  const [category, setCategory] = React.useState<string>('');
  const [accessDenied, setAccessDenied] = React.useState(false);
  const [active, setActive] = React.useState<Permission | null>(null);

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const res = await Api.rbac.getPermissionList({ page: 1, pageSize: 3000 });
      setList(res.data.list || []);
      setAccessDenied(false);
    } catch (err) {
      if (err instanceof ApiRequestError && (err.statusCode === 403 || err.businessCode === 2004)) {
        setAccessDenied(true);
        return;
      }
      message.error(err instanceof Error ? err.message : '加载权限失败');
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  const categories = React.useMemo(() => {
    const set = new Set(list.map((item) => item.category).filter(Boolean));
    return Array.from(set).sort();
  }, [list]);

  const filtered = list.filter((item) => {
    const q = query.trim().toLowerCase();
    const matchQuery = q
      ? [item.code, item.name, item.description, item.category]
          .some((value) => String(value || '').toLowerCase().includes(q))
      : true;
    const matchCategory = category ? item.category === category : true;
    return matchQuery && matchCategory;
  });

  const copyPermissionCode = async (code: string) => {
    try {
      await navigator.clipboard.writeText(code);
      message.success('权限 Code 已复制');
    } catch {
      message.warning('当前环境不支持自动复制，请手动复制');
    }
  };

  if (accessDenied) {
    return <AccessDeniedPage />;
  }

  return (
    <Card
      title="权限管理"
      extra={(
        <Space wrap>
          <Input
            allowClear
            aria-label="搜索权限"
            placeholder="搜索权限 code/名称/分类"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            style={{ width: 280 }}
          />
          <Select
            allowClear
            value={category || undefined}
            placeholder="按分组筛选"
            style={{ width: 180 }}
            options={categories.map((value) => ({ value, label: value }))}
            onChange={(value) => setCategory(value || '')}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>刷新</Button>
        </Space>
      )}
    >
      <Table
        rowKey="id"
        loading={loading}
        locale={{ emptyText: <Empty description="暂无权限数据" /> }}
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
          'aria-label': `查看权限 ${record.code} 详情`,
        })}
        columns={[
          { title: 'Code', dataIndex: 'code', render: (value: string) => <code>{value}</code> },
          { title: '名称', dataIndex: 'name' },
          { title: '分类', dataIndex: 'category', render: (value: string) => <Tag>{value || '-'}</Tag> },
          { title: '描述', dataIndex: 'description' },
          {
            title: '操作',
            width: 180,
            render: (_: unknown, row: Permission) => (
              <Space>
                <Button
                  type="link"
                  onClick={(event) => {
                    event.stopPropagation();
                    setActive(row);
                  }}
                  aria-label={`查看权限 ${row.code} 详情`}
                >
                  详情
                </Button>
                <Button
                  type="link"
                  icon={<CopyOutlined />}
                  onClick={(event) => {
                    event.stopPropagation();
                    void copyPermissionCode(row.code);
                  }}
                  aria-label={`复制权限 ${row.code}`}
                >
                  复制
                </Button>
              </Space>
            ),
          },
        ]}
      />

      <Drawer title="权限详情" open={Boolean(active)} onClose={() => setActive(null)} width={420}>
        {!active ? null : (
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <div><strong>Code：</strong><code>{active.code}</code></div>
            <div><strong>名称：</strong>{active.name}</div>
            <div><strong>分类：</strong>{active.category || '-'}</div>
            <div><strong>描述：</strong>{active.description || '-'}</div>
          </Space>
        )}
      </Drawer>
    </Card>
  );
};

export default PermissionsPage;
