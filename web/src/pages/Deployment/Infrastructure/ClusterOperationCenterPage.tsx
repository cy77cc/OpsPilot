import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ArrowLeftOutlined,
  ReloadOutlined,
  SearchOutlined,
  EyeOutlined,
  ClusterOutlined,
} from '@ant-design/icons';
import {
  Button,
  Card,
  DatePicker,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  message,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { Api } from '../../../api';
import { DetailSkeleton, TableSkeleton } from '../../../components/LoadingSkeleton';
import { GuidedFormItem } from '../../../components/FormGuidance';
import type {
  Cluster,
  ClusterOperationDetail,
  ClusterOperationHistoryItem,
  ClusterOperationState,
} from '../../../api/modules/cluster';

const { RangePicker } = DatePicker;
const { Text, Paragraph, Title } = Typography;

type OperationFilters = {
  resource?: string;
  status?: string;
  operator?: string;
  from?: string;
  to?: string;
};

type OperationTraceRecord = ClusterOperationHistoryItem & Partial<{
  resource_id?: string | number;
  request?: Record<string, unknown>;
  response?: Record<string, unknown>;
}>;

type PolicyReleaseTrace = {
  releaseId?: string;
  version?: string;
  policyName?: string;
  namespace?: string;
};

const POLICY_RELEASE_RESOURCE = 'policy_release';

const statusMeta: Record<ClusterOperationState | string, { color: string; text: string }> = {
  completed: { color: 'green', text: '已完成' },
  approval_required: { color: 'orange', text: '待审批' },
  rejected: { color: 'default', text: '已拒绝' },
  failed: { color: 'red', text: '失败' },
};

const resourceOptions = [
  { value: 'node', label: '节点' },
  { value: 'deployment', label: 'Deployment' },
  { value: 'statefulset', label: 'StatefulSet' },
  { value: 'pod', label: 'Pod' },
  { value: 'service', label: 'Service' },
  { value: 'ingress', label: 'Ingress' },
  { value: 'cluster', label: '集群' },
  { value: 'certificate', label: '证书' },
  { value: POLICY_RELEASE_RESOURCE, label: '策略发布' },
];

function formatDate(value?: string) {
  if (!value) {return '-';}
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss');
}

function buildQuery(filters: OperationFilters, page: number, pageSize: number) {
  return {
    page,
    page_size: pageSize,
    resource: filters.resource || undefined,
    status: filters.status || undefined,
    operator: filters.operator || undefined,
    from: filters.from || undefined,
    to: filters.to || undefined,
  };
}

function formatSummaryValue(value: unknown): string {
  if (value === null || value === undefined || value === '') {return '-';}
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return String(value);
  }
  if (Array.isArray(value)) {
    return value.length ? value.map((entry) => formatSummaryValue(entry)).join(', ') : '-';
  }
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function summarizeObject(payload?: Record<string, unknown> | null) {
  if (!payload) {return '-';}
  const entries = Object.entries(payload).filter(([, value]) => value !== undefined);
  if (entries.length === 0) {return '-';}
  return entries.slice(0, 4).map(([key, value]) => `${key}: ${formatSummaryValue(value)}`).join(' | ');
}

function summarizeDiagnostics(items?: unknown[] | null) {
  if (!items?.length) {return '-';}
  return items.slice(0, 3).map((item) => {
    if (item && typeof item === 'object' && !Array.isArray(item)) {
      const record = item as Record<string, unknown>;
      const level = formatSummaryValue(record.level);
      const messageText = formatSummaryValue(record.message);
      if (level !== '-' || messageText !== '-') {
        return [level !== '-' ? level : '', messageText !== '-' ? messageText : '']
          .filter(Boolean)
          .join(': ');
      }
    }
    return formatSummaryValue(item);
  }).join(' | ');
}

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function normalizeTraceText(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value);
  }
  if (typeof value === 'string' && value.trim()) {
    return value.trim();
  }
  return undefined;
}

function extractPolicyReleaseTrace(record?: OperationTraceRecord | ClusterOperationDetail | null): PolicyReleaseTrace {
  if (!record) {
    return {};
  }

  const source = record as OperationTraceRecord;
  const requestRelease = asRecord(asRecord(source.request)?.release);
  const responseRelease = asRecord(asRecord(source.response)?.release);
  const release = responseRelease || requestRelease;
  const policy = asRecord(release?.policy) || asRecord(requestRelease?.policy) || asRecord(responseRelease?.policy);

  return {
    releaseId: normalizeTraceText(source.resource_id)
      || normalizeTraceText(release?.release_id)
      || normalizeTraceText(release?.releaseId),
    version: normalizeTraceText(source.target)
      || normalizeTraceText(release?.version),
    policyName: normalizeTraceText(source.resource_name)
      || normalizeTraceText(policy?.name),
    namespace: normalizeTraceText(source.namespace)
      || normalizeTraceText(policy?.namespace),
  };
}

function parseFilters(searchParams: URLSearchParams): OperationFilters {
  const releaseId = normalizeTraceText(searchParams.get('release_id'));
  const resource = normalizeTraceText(searchParams.get('resource')) || (releaseId ? POLICY_RELEASE_RESOURCE : undefined);
  return {
    resource,
    status: normalizeTraceText(searchParams.get('status')),
    operator: normalizeTraceText(searchParams.get('operator')),
    from: normalizeTraceText(searchParams.get('from')),
    to: normalizeTraceText(searchParams.get('to')),
  };
}

function buildRangeValue(filters: OperationFilters): [dayjs.Dayjs | null, dayjs.Dayjs | null] | undefined {
  if (!filters.from && !filters.to) {
    return undefined;
  }
  return [
    filters.from ? dayjs(filters.from) : null,
    filters.to ? dayjs(filters.to) : null,
  ];
}

const ClusterOperationCenterPage: React.FC = () => {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const clusterId = Number(id);
  const [searchParams, setSearchParams] = useSearchParams();
  const [filterForm] = Form.useForm();

  const [cluster, setCluster] = useState<Cluster | null>(null);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [history, setHistory] = useState<ClusterOperationHistoryItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [filters, setFilters] = useState<OperationFilters>({});
  const [selectedAuditId, setSelectedAuditId] = useState<string>('');
  const [selectedDetail, setSelectedDetail] = useState<ClusterOperationDetail | null>(null);
  const releaseFilterId = normalizeTraceText(searchParams.get('release_id'));
  const searchFilters = useMemo(() => parseFilters(searchParams), [searchParams]);

  const buildOperationLink = useCallback((auditId?: string | number, releaseId?: string) => {
    const params = new URLSearchParams();
    if (searchFilters.resource) {params.set('resource', searchFilters.resource);}
    if (searchFilters.status) {params.set('status', searchFilters.status);}
    if (searchFilters.operator) {params.set('operator', searchFilters.operator);}
    if (searchFilters.from) {params.set('from', searchFilters.from);}
    if (searchFilters.to) {params.set('to', searchFilters.to);}
    if (releaseId) {params.set('release_id', releaseId);}
    if (auditId) {params.set('audit_id', String(auditId));}
    const query = params.toString();
    return `/deployment/infrastructure/clusters/${clusterId}/operations${query ? `?${query}` : ''}`;
  }, [clusterId, searchFilters.from, searchFilters.operator, searchFilters.resource, searchFilters.status, searchFilters.to]);

  const loadCluster = useCallback(async () => {
    if (!clusterId) {return;}
    try {
      const res = await Api.cluster.getClusterDetail(clusterId);
      setCluster(res.data);
    } catch {
      setCluster(null);
    }
  }, [clusterId]);

  const loadHistory = useCallback(async (nextPage: number, nextPageSize: number, nextFilters: OperationFilters) => {
    if (!clusterId) {return;}
    setLoading(true);
    try {
      const res = await Api.cluster.getClusterOperations(clusterId, buildQuery(nextFilters, nextPage, nextPageSize));
      const responseMeta = res.data as unknown as Record<string, unknown>;
      const pageCandidate = Number(responseMeta.page);
      const pageSizeCandidate = Number(responseMeta.page_size);
      const resolvedPage = Number.isFinite(pageCandidate) && pageCandidate > 0 ? pageCandidate : nextPage;
      const resolvedPageSize = Number.isFinite(pageSizeCandidate) && pageSizeCandidate > 0 ? pageSizeCandidate : nextPageSize;
      setHistory(res.data.list || []);
      setTotal(res.data.total || 0);
      setPage(resolvedPage);
      setPageSize(resolvedPageSize);
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载操作历史失败');
      setHistory([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  const loadDetail = useCallback(async (auditId: string | number) => {
    if (!clusterId || !auditId) {return;}
    const auditIDText = String(auditId);
    setSelectedAuditId(auditIDText);
    setDetailLoading(true);
    try {
      const res = await Api.cluster.getClusterOperationDetail(clusterId, auditIDText);
      setSelectedDetail(res.data);
      setDetailOpen(true);
      const trace = extractPolicyReleaseTrace(res.data);
      const nextParams = new URLSearchParams(searchParams);
      nextParams.set('audit_id', auditIDText);
      if (trace.releaseId) {
        nextParams.set('resource', POLICY_RELEASE_RESOURCE);
        nextParams.set('release_id', trace.releaseId);
      }
      setSearchParams(nextParams, { replace: true });
    } catch (err) {
      message.error(err instanceof Error ? err.message : '加载操作详情失败');
    } finally {
      setDetailLoading(false);
    }
  }, [clusterId, searchParams, setSearchParams]);

  useEffect(() => {
    void loadCluster();
  }, [loadCluster]);

  useEffect(() => {
    setFilters(searchFilters);
    filterForm.setFieldsValue({
      resource: searchFilters.resource,
      status: searchFilters.status,
      operator: searchFilters.operator,
      range: buildRangeValue(searchFilters),
    });
    void loadHistory(1, 20, searchFilters);
  }, [clusterId, filterForm, loadHistory, searchFilters]);

  useEffect(() => {
    const auditId = searchParams.get('audit_id');
    if (auditId && auditId !== selectedAuditId) {
      setSelectedAuditId(auditId);
      void loadDetail(auditId);
    }
  }, [loadDetail, searchParams, selectedAuditId]);

  const visibleHistory = useMemo(() => {
    if (!releaseFilterId) {
      return history;
    }
    return history.filter((item) => extractPolicyReleaseTrace(item as OperationTraceRecord).releaseId === releaseFilterId);
  }, [history, releaseFilterId]);

  useEffect(() => {
    if (!releaseFilterId || searchParams.get('audit_id') || detailLoading || selectedAuditId) {
      return;
    }
    const matched = visibleHistory[0];
    if (matched) {
      void loadDetail(matched.audit_id);
    }
  }, [detailLoading, loadDetail, releaseFilterId, searchParams, selectedAuditId, visibleHistory]);

  const submitFilters = async (values: { resource?: string; status?: string; operator?: string; range?: [dayjs.Dayjs | null, dayjs.Dayjs | null] }) => {
    const nextFilters: OperationFilters = {
      resource: values.resource || undefined,
      status: values.status || undefined,
      operator: values.operator?.trim() || undefined,
      from: values.range?.[0] ? values.range[0].toISOString() : undefined,
      to: values.range?.[1] ? values.range[1].toISOString() : undefined,
    };
    setFilters(nextFilters);
    setPage(1);
    const nextParams = new URLSearchParams(searchParams);
    if (nextFilters.resource) {nextParams.set('resource', nextFilters.resource);}
    else {nextParams.delete('resource');}
    if (nextFilters.status) {nextParams.set('status', nextFilters.status);}
    else {nextParams.delete('status');}
    if (nextFilters.operator) {nextParams.set('operator', nextFilters.operator);}
    else {nextParams.delete('operator');}
    if (nextFilters.from) {nextParams.set('from', nextFilters.from);}
    else {nextParams.delete('from');}
    if (nextFilters.to) {nextParams.set('to', nextFilters.to);}
    else {nextParams.delete('to');}
    if (releaseFilterId && nextFilters.resource === POLICY_RELEASE_RESOURCE) {
      nextParams.set('release_id', releaseFilterId);
    } else {
      nextParams.delete('release_id');
    }
    setSearchParams(nextParams, { replace: true });
    await loadHistory(1, pageSize, nextFilters);
  };

  const resetFilters = async () => {
    filterForm.resetFields();
    setFilters({});
    setPage(1);
    const nextParams = new URLSearchParams(searchParams);
    nextParams.delete('resource');
    nextParams.delete('status');
    nextParams.delete('operator');
    nextParams.delete('from');
    nextParams.delete('to');
    nextParams.delete('release_id');
    setSearchParams(nextParams, { replace: true });
    await loadHistory(1, pageSize, {});
  };

  const columns: ColumnsType<ClusterOperationHistoryItem> = useMemo(() => [
    {
      title: 'Audit ID',
      dataIndex: 'audit_id',
      key: 'audit_id',
      render: (auditId: string | number) => (
        <Button type="link" className="px-0" onClick={() => void loadDetail(auditId)}>
          {auditId}
        </Button>
      ),
    },
    {
      title: '操作',
      dataIndex: 'action',
      key: 'action',
      render: (action: string) => <Tag color="blue">{action}</Tag>,
    },
    {
      title: '资源',
      key: 'resource',
      render: (_, record) => (
        <div>
          {(() => {
            const trace = extractPolicyReleaseTrace(record as OperationTraceRecord);
            return (
              <>
                <div>{trace.policyName || record.resource_name || record.target || record.resource || record.resource_type || '-'}</div>
                <Space size={8} wrap>
                  <Text type="secondary">{trace.namespace || record.namespace || record.resource_type || '-'}</Text>
                  {trace.releaseId ? (
                    <Link to={buildOperationLink(record.audit_id, trace.releaseId)}>
                      Release #{trace.releaseId}
                    </Link>
                  ) : null}
                  {trace.version ? <Text type="secondary">{trace.version}</Text> : null}
                </Space>
              </>
            );
          })()}
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const meta = statusMeta[status] || { color: 'default', text: status };
        return <Tag color={meta.color}>{meta.text}</Tag>;
      },
    },
    {
      title: '操作人',
      dataIndex: 'operator',
      key: 'operator',
      render: (operator?: string) => operator || '-',
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time: string) => formatDate(time),
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
      render: (messageText?: string) => messageText || '-',
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_, record) => (
        <Space>
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => void loadDetail(record.audit_id)}>
            详情
          </Button>
        </Space>
      ),
    },
  ], [buildOperationLink, loadDetail]);

  const detail = selectedDetail;
  const detailApproval = detail?.approval;
  const detailAuditLink = selectedAuditId || detail?.audit_id;
  const detailTrace = extractPolicyReleaseTrace(detail as OperationTraceRecord | null);
  const isInitialLoading = loading && history.length === 0;
  const detailApprovalRequired = detailApproval?.required ?? Boolean(detailApproval?.ticket || detail?.status === 'approval_required');
  const requestSummary = summarizeObject(detail?.request);
  const responseSummary = summarizeObject(detail?.response);
  const diagnosticsSummary = summarizeDiagnostics(detail?.diagnostics);
  const effectiveTotal = releaseFilterId ? visibleHistory.length : total;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div className="flex items-center gap-4">
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/deployment/infrastructure/clusters/${clusterId}`)}>
            返回详情
          </Button>
          <div>
            <Title level={4} className="m-0 flex items-center gap-2">
              <ClusterOutlined />
              {cluster?.name || `集群 #${clusterId}`} 操作中心
            </Title>
            <Text type="secondary">查看高风险操作审计、审批状态与执行详情</Text>
            {releaseFilterId ? (
              <div>
                <Tag color="purple">Policy Release #{releaseFilterId}</Tag>
              </div>
            ) : null}
          </div>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void loadHistory(page, pageSize, filters)} loading={loading && !isInitialLoading}>
            刷新
          </Button>
          <Button onClick={() => navigate(`/deployment/infrastructure/clusters/${clusterId}`)}>回到集群详情</Button>
        </Space>
      </div>

      <Card>
        <Form
          form={filterForm}
          layout="inline"
          onFinish={submitFilters}
          initialValues={{ resource: filters.resource, status: filters.status, operator: filters.operator }}
        >
          <Space size={12} wrap>
            <Form.Item name="resource" label="资源">
              <Select
                allowClear
                placeholder="全部"
                style={{ width: 160 }}
                options={resourceOptions}
              />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select
                allowClear
                placeholder="全部"
                style={{ width: 160 }}
                options={[
                  { value: 'completed', label: '已完成' },
                  { value: 'approval_required', label: '待审批' },
                  { value: 'rejected', label: '已拒绝' },
                  { value: 'failed', label: '失败' },
                ]}
              />
            </Form.Item>
            <GuidedFormItem name="operator" label="操作人">
              <Input placeholder="用户 ID / 名称" style={{ width: 180 }} allowClear />
            </GuidedFormItem>
            <Form.Item label="时间范围" name="range">
              <DatePicker.RangePicker showTime />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" htmlType="submit" icon={<SearchOutlined />}>
                  查询
                </Button>
                <Button onClick={() => void resetFilters()}>重置</Button>
              </Space>
            </Form.Item>
          </Space>
        </Form>
      </Card>

      <Card>
        {isInitialLoading ? (
          <TableSkeleton toolbar={false} rows={10} columns={7} />
        ) : (
          <Table
            dataSource={visibleHistory}
            columns={columns}
            rowKey="audit_id"
            loading={false}
            pagination={{
              current: page,
              pageSize,
              total: effectiveTotal,
              showSizeChanger: true,
              showTotal: (value) => `共 ${value} 条`,
              onChange: (nextPage, nextSize) => {
                void loadHistory(nextPage, nextSize || pageSize, filters);
              },
            }}
          />
        )}
      </Card>

      <Drawer
        title={detail ? `操作详情: ${detail.audit_id}` : '操作详情'}
        width={720}
        open={detailOpen}
        onClose={() => {
          setDetailOpen(false);
          setSelectedDetail(null);
        }}
        extra={detailAuditLink ? (
          <Button type="link" onClick={() => navigate(buildOperationLink(detailAuditLink, detailTrace.releaseId))}>
            复制/共享当前审计链接
          </Button>
        ) : null}
      >
        {detailLoading ? (
          <DetailSkeleton summaryCards={1} sections={3} />
        ) : detail ? (
          <div className="space-y-5">
            <Descriptions bordered size="small" column={1}>
              <Descriptions.Item label="Audit ID">{detail.audit_id}</Descriptions.Item>
              <Descriptions.Item label="动作">{detail.action}</Descriptions.Item>
              <Descriptions.Item label="资源">
                {detail.resource_name || detail.target || detail.resource || detail.resource_type || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="资源类型">{detail.resource_type || detail.resource || '-'}</Descriptions.Item>
              {detailTrace.releaseId ? (
                <Descriptions.Item label="Release ID">#{detailTrace.releaseId}</Descriptions.Item>
              ) : null}
              {detailTrace.version ? (
                <Descriptions.Item label="发布版本">{detailTrace.version}</Descriptions.Item>
              ) : null}
              <Descriptions.Item label="状态">
                <Tag color={(statusMeta[detail.status] || { color: 'default' }).color}>
                  {(statusMeta[detail.status] || { text: detail.status }).text}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="操作人">{detail.operator || '-'}</Descriptions.Item>
              <Descriptions.Item label="消息">{detail.message || '-'}</Descriptions.Item>
              <Descriptions.Item label="创建时间">{formatDate(detail.created_at)}</Descriptions.Item>
            </Descriptions>

            {detailApproval && (
              <Card size="small" title="审批信息">
                <Descriptions bordered size="small" column={1}>
                  <Descriptions.Item label="是否需要审批">{detailApprovalRequired ? '是' : '否'}</Descriptions.Item>
                  <Descriptions.Item label="审批状态">{detailApproval.status || '-'}</Descriptions.Item>
                  <Descriptions.Item label="审批票据">{detailApproval.ticket || '-'}</Descriptions.Item>
                  <Descriptions.Item label="过期时间">{formatDate(detailApproval.expires_at)}</Descriptions.Item>
                  <Descriptions.Item label="原因">{detailApproval.reason || '-'}</Descriptions.Item>
                  <Descriptions.Item label="已消费时间">{formatDate(detailApproval.consumed_at)}</Descriptions.Item>
                  <Descriptions.Item label="重放信息">
                    {detailApproval.replay_count
                      ? `${detailApproval.replay_code || '-'} | ${detailApproval.replay_message || '-'}`
                      : '-'}
                  </Descriptions.Item>
                </Descriptions>
              </Card>
            )}

            <Card size="small" title="请求摘要">
              <Descriptions bordered size="small" column={1} className="mb-3">
                <Descriptions.Item label="请求摘要">{requestSummary}</Descriptions.Item>
              </Descriptions>
              {detail.request ? (
                <pre className="bg-gray-950 text-gray-100 rounded p-4 overflow-auto max-h-80 text-xs">
                  {JSON.stringify(detail.request, null, 2)}
                </pre>
              ) : (
                <Empty description="暂无请求摘要" />
              )}
            </Card>

            <Card size="small" title="响应摘要">
              <Descriptions bordered size="small" column={1} className="mb-3">
                <Descriptions.Item label="响应摘要">{responseSummary}</Descriptions.Item>
              </Descriptions>
              {detail.response ? (
                <pre className="bg-gray-950 text-gray-100 rounded p-4 overflow-auto max-h-80 text-xs">
                  {JSON.stringify(detail.response, null, 2)}
                </pre>
              ) : (
                <Empty description="暂无响应摘要" />
              )}
            </Card>

            <Card size="small" title="诊断摘要">
              {detail.diagnostics?.length ? (
                <>
                  <Descriptions bordered size="small" column={1} className="mb-3">
                    <Descriptions.Item label="诊断摘要">{diagnosticsSummary}</Descriptions.Item>
                  </Descriptions>
                  <pre className="bg-gray-950 text-gray-100 rounded p-4 overflow-auto max-h-80 text-xs">
                    {JSON.stringify(detail.diagnostics, null, 2)}
                  </pre>
                </>
              ) : (
                <Empty description="暂无诊断摘要" />
              )}
            </Card>

            {detail.timeline?.length ? (
              <Card size="small" title="时间线">
                <Space direction="vertical" className="w-full" size={12}>
                  {detail.timeline.map((item, index) => (
                    <div key={`${item.at || index}`} className="border rounded p-3">
                      <div className="flex items-center justify-between gap-3">
                        <Text strong>{item.status || item.level || 'event'}</Text>
                        <Text type="secondary">{formatDate(item.at)}</Text>
                      </div>
                      <Paragraph className="mb-0 mt-2">{item.message || '-'}</Paragraph>
                    </div>
                  ))}
                </Space>
              </Card>
            ) : null}
          </div>
        ) : (
          <Empty description="未选择操作" />
        )}
      </Drawer>
    </div>
  );
};

export default ClusterOperationCenterPage;
