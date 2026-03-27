import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Alert, Button, Space, Tag, Collapse, Empty, message } from 'antd';
import {
  ReloadOutlined,
  WarningOutlined,
  CheckCircleOutlined,
  BulbOutlined,
  LineChartOutlined,
} from '@ant-design/icons';
import { Api } from '../../../api';
import type { RiskFinding, Anomaly, Suggestion } from '../../../api/modules/aiops';

const { Panel } = Collapse;

const AIOpsInsightsPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [riskFindings, setRiskFindings] = useState<RiskFinding[]>([]);
  const [anomalies, setAnomalies] = useState<Anomaly[]>([]);
  const [suggestions, setSuggestions] = useState<Suggestion[]>([]);

  const load = async () => {
    setLoading(true);
    try {
      const [risksRes, anomaliesRes, suggestionsRes] = await Promise.all([
        Api.aiops.getRiskFindings(),
        Api.aiops.getAnomalies(),
        Api.aiops.getSuggestions(),
      ]);
      setRiskFindings(risksRes.data.list || []);
      setAnomalies(anomaliesRes.data.list || []);
      setSuggestions(suggestionsRes.data.list || []);
    } catch (err) {
      message.error('加载 AIOps 数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
      case 'high':
        return 'error';
      case 'medium':
        return 'warning';
      case 'low':
        return 'success';
      default:
        return 'default';
    }
  };

  const getSeverityLabel = (severity: string) => {
    const labels: Record<string, string> = {
      critical: '严重',
      high: '高',
      medium: '中',
      low: '低',
    };
    return labels[severity] || severity;
  };

  const getImpactColor = (impact: string) => {
    switch (impact) {
      case 'high':
        return 'red';
      case 'medium':
        return 'orange';
      case 'low':
        return 'green';
      default:
        return 'default';
    }
  };

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">AIOps 洞察</h1>
          <p className="text-sm text-gray-500 mt-1">智能运维分析与优化建议</p>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={load} loading={loading}>
            刷新
          </Button>
        </Space>
      </div>

      {/* Summary cards */}
      <Row gutter={[16, 16]}>
        <Col xs={24} md={8}>
          <Card>
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-500">风险发现</div>
                <div className="text-2xl font-semibold text-red-500">{riskFindings.length}</div>
              </div>
              <WarningOutlined className="text-3xl text-red-400" />
            </div>
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card>
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-500">异常检测</div>
                <div className="text-2xl font-semibold text-orange-500">{anomalies.length}</div>
              </div>
              <LineChartOutlined className="text-3xl text-orange-400" />
            </div>
          </Card>
        </Col>
        <Col xs={24} md={8}>
          <Card>
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-500">优化建议</div>
                <div className="text-2xl font-semibold text-green-500">{suggestions.length}</div>
              </div>
              <BulbOutlined className="text-3xl text-green-400" />
            </div>
          </Card>
        </Col>
      </Row>

      {/* Risk Findings */}
      <Card title="风险发现">
        {riskFindings.length > 0 ? (
          <Collapse>
            {riskFindings.map((risk, index) => (
              <Panel
                key={index}
                header={
                  <Space>
                    <Tag color={getSeverityColor(risk.severity)}>{getSeverityLabel(risk.severity)}</Tag>
                    <span>{risk.title}</span>
                    <span className="text-gray-400 text-sm">{risk.service_name}</span>
                  </Space>
                }
              >
                <div className="space-y-2">
                  <div>
                    <strong>类型:</strong> {risk.type}
                  </div>
                  <div>
                    <strong>描述:</strong> {risk.description}
                  </div>
                  <div className="text-xs text-gray-500">
                    发现时间: {new Date(risk.created_at).toLocaleString()}
                  </div>
                </div>
              </Panel>
            ))}
          </Collapse>
        ) : (
          <Empty description="暂无风险发现" />
        )}
      </Card>

      {/* Anomalies */}
      <Card title="异常检测">
        {anomalies.length > 0 ? (
          <div className="space-y-4">
            {anomalies.map((anomaly) => (
              <Alert
                key={anomaly.id}
                type="warning"
                showIcon
                message={`${anomaly.service_name}: ${anomaly.metric}`}
                description={
                  <div>
                    <div>当前值: {anomaly.value} | 阈值: {anomaly.threshold}</div>
                    <div className="text-xs text-gray-500">
                      检测时间: {new Date(anomaly.detected_at).toLocaleString()}
                    </div>
                  </div>
                }
              />
            ))}
          </div>
        ) : (
          <Empty description="暂无异常检测" />
        )}
      </Card>

      {/* Suggestions */}
      <Card title="优化建议">
        {suggestions.length > 0 ? (
          <Row gutter={[16, 16]}>
            {suggestions.map((suggestion) => (
              <Col xs={24} md={8} key={suggestion.id}>
                <Card className="h-full" size="small">
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <Tag color={getImpactColor(suggestion.impact)}>{suggestion.impact}影响</Tag>
                      <span className="text-xs text-gray-400">{suggestion.type}</span>
                    </div>
                    <div className="font-semibold">{suggestion.title}</div>
                    <div className="text-sm text-gray-600">{suggestion.description}</div>
                    {suggestion.service_name && (
                      <div className="text-xs text-gray-400">服务: {suggestion.service_name}</div>
                    )}
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        ) : (
          <Empty description="暂无优化建议" />
        )}
      </Card>
    </div>
  );
};

export default AIOpsInsightsPage;
