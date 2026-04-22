import type { FieldGuide } from '../components/FormGuidance';

export const commonFieldGuides: Record<string, FieldGuide> = {
  name: {
    whatToEnter: '给此资源起一个易于识别的显示名称。',
    purpose: '用于在列表和详情页面展示，方便团队沟通。',
    example: 'prod-cluster-01, web-server-staging',
    aiPlaceholder: "例如：'生成一个包含环境和地域的集群名称'",
  },
  description: {
    whatToEnter: '对此资源的用途、状态或注意事项进行说明。',
    purpose: '帮助其他团队成员理解该资源的背景，减少沟通成本。',
    example: '该集群用于北京地域的生产环境 Web 服务。',
    aiPlaceholder: "例如：'描述一个高可用的北京生产环境集群'",
  },
  cron: {
    whatToEnter: '使用标准的 Cron 表达式定义定时任务的触发频率。',
    purpose: '控制任务在特定的时间点（如每天凌晨、每小时等）自动运行。',
    example: '0 0 * * * (每天凌晨), */30 * * * * (每30分钟)',
    aiPlaceholder: "例如：'每天凌晨两点运行一次'",
  },
  json: {
    whatToEnter: '输入有效的 JSON 格式配置项。',
    purpose: '为资源提供灵活的参数配置或复杂的策略定义。',
    example: '{ "replicas": 3, "strategy": "rolling" }',
    aiPlaceholder: "例如：'生成一个 3 副本的滚动更新配置'",
  },
};

export const monitorFieldGuides: Record<string, FieldGuide> = {
  promqlExpr: {
    whatToEnter: '输入完整的 Prometheus 查询语句 (PromQL)。',
    purpose: '定义告警的计算逻辑，支持复杂的数学运算、函数和过滤条件。',
    example: 'node_cpu_seconds_total{mode="idle"} > 0.9, http_requests_total{status="500"} > 10',
    aiPlaceholder: "例如：'计算过去5分钟内，5xx请求数超过100的语句'",
  },
  durationSec: {
    whatToEnter: '定义告警持续多久才触发。',
    purpose: '防止告警抖动，仅在指标持续异常超过该时长后发送通知。',
    example: '300 (5分钟), 60 (1分钟)',
    aiPlaceholder: "例如：'告警触发前保持异常状态持续5分钟'",
  },
  labelsJson: {
    whatToEnter: '输入 JSON 格式的附加标签，用于给告警打标。',
    purpose: '方便告警路由、抑制、聚合以及后续的查询分析。',
    example: '{"team": "sre", "env": "prod"}',
    aiPlaceholder: "例如：'为告警添加团队和环境标签'",
  },
  annotationsJson: {
    whatToEnter: '输入 JSON 格式的告警详情注解。',
    purpose: '提供告警的摘要、详细描述和排查建议，直接显示在通知中。',
    example: '{"summary": "CPU利用率过高", "description": "节点CPU持续高于90%"}',
    aiPlaceholder: "例如：'生成告警的摘要和排查建议'",
  },
};
