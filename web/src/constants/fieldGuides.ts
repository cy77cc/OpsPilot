import type { FieldGuide } from '../components/FormGuidance';

export const commonFieldGuides: Record<string, FieldGuide> = {
  name: {
    whatToEnter: '给此资源起一个易于识别的显示名称。',
    purpose: '用于在列表和详情页面展示，方便团队沟通。',
    example: 'prod-cluster-01, web-server-staging',
  },
  description: {
    whatToEnter: '对此资源的用途、状态或注意事项进行说明。',
    purpose: '帮助其他团队成员理解该资源的背景，减少沟通成本。',
    example: '该集群用于北京地域的生产环境 Web 服务。',
  },
  cron: {
    whatToEnter: '使用标准的 Cron 表达式定义定时任务的触发频率。',
    purpose: '控制任务在特定的时间点（如每天凌晨、每小时等）自动运行。',
    example: '0 0 * * * (每天凌晨), */30 * * * * (每30分钟)',
  },
  json: {
    whatToEnter: '输入有效的 JSON 格式配置项。',
    purpose: '为资源提供灵活的参数配置或复杂的策略定义。',
    example: '{ "replicas": 3, "strategy": "rolling" }',
  },
};
